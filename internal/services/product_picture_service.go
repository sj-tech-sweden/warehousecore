package services

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"

	"warehousecore/internal/services/storage"
)

// ProductPictureService handles uploads and retrieval of product pictures in Nextcloud.
type ProductPictureService struct {
	client       *storage.NextcloudClient
	enabled      bool
	maxFileSize  int64
	allowedTypes map[string]bool
	rootFolder   string
	cacheDir     string
}

type ProductPictureInfo struct {
	FileName    string    `json:"file_name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	ModifiedAt  time.Time `json:"modified_at"`
}

// NewProductPictureServiceFromEnv builds the service if Nextcloud credentials are present.
func NewProductPictureServiceFromEnv() *ProductPictureService {
	ncURL := trimAndUnquote(os.Getenv("NEXTCLOUD_WEBDAV_URL"))
	ncUser := trimAndUnquote(os.Getenv("NEXTCLOUD_WEBDAV_USER"))
	ncPass := trimAndUnquote(os.Getenv("NEXTCLOUD_WEBDAV_PASSWORD"))
	ncBase := trimAndUnquote(os.Getenv("NEXTCLOUD_WEBDAV_BASE_PATH"))
	cacheDir := trimAndUnquote(os.Getenv("PICTURE_CACHE_DIR"))
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "warehousecore", "pictures_cache")
	}

	service := &ProductPictureService{
		enabled:     false,
		maxFileSize: 10 * 1024 * 1024, // 10 MB
		allowedTypes: map[string]bool{
			"image/jpeg": true,
			"image/jpg":  true,
			"image/png":  true,
			"image/gif":  true,
			"image/webp": true,
			"image/heic": true,
		},
		rootFolder: path.Join("warehousecore", "pictures"),
		cacheDir:   cacheDir,
	}

	if ncURL == "" || ncUser == "" || ncPass == "" {
		log.Println("[PICTURES] Nextcloud credentials missing - product pictures disabled")
		return service
	}

	client, err := storage.NewNextcloudClient(ncURL, ncUser, ncPass, ncBase)
	if err != nil {
		log.Printf("[PICTURES] Failed to configure Nextcloud client: %v\n", err)
		return service
	}

	service.client = client
	service.enabled = true
	log.Printf("[PICTURES] Nextcloud product pictures enabled at base '%s'\n", service.rootFolder)
	return service
}

// Enabled reports whether uploads are available.
func (s *ProductPictureService) Enabled() bool {
	return s != nil && s.enabled && s.client != nil
}

// MaxFileSize returns the allowed upload size in bytes.
func (s *ProductPictureService) MaxFileSize() int64 {
	if s == nil {
		return 0
	}
	return s.maxFileSize
}

// FolderForProduct returns the sanitized folder path (relative to the Nextcloud base).
func (s *ProductPictureService) FolderForProduct(productName string) string {
	return s.productFolder(productName)
}

// UploadPicture saves a new image for the given product name and returns the stored filename.
func (s *ProductPictureService) UploadPicture(productName string, file multipart.File, header *multipart.FileHeader) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("product pictures not configured")
	}

	defer file.Seek(0, io.SeekStart) // ensure caller can reuse/close safely

	buf := bytes.Buffer{}
	limitedReader := io.LimitReader(file, s.maxFileSize+1)
	if _, err := io.Copy(&buf, limitedReader); err != nil {
		return "", fmt.Errorf("read upload: %w", err)
	}
	if int64(buf.Len()) > s.maxFileSize {
		return "", fmt.Errorf("file exceeds limit of %d bytes", s.maxFileSize)
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" && buf.Len() > 0 {
		contentType = http.DetectContentType(buf.Bytes())
	}
	if !s.allowedTypes[contentType] {
		return "", fmt.Errorf("unsupported file type: %s", contentType)
	}

	filename := sanitizeFileName(header.Filename)
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := strings.ToLower(filepath.Ext(filename))
	if base == "" {
		base = "bild"
	}
	storedName := fmt.Sprintf("%s_%d%s", base, time.Now().Unix(), ext)

	relPath := s.productFilePath(productName, storedName)
	if err := s.client.Upload(relPath, bytes.NewReader(buf.Bytes()), contentType); err != nil {
		return "", fmt.Errorf("upload to nextcloud: %w", err)
	}

	return storedName, nil
}

// ListPictures returns all files for a product.
func (s *ProductPictureService) ListPictures(productName string) ([]ProductPictureInfo, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("product pictures not configured")
	}
	folder := s.productFolder(productName)
	// Ensure folder hierarchy exists so PROPFIND does not 404 on first access.
	_ = s.client.EnsureCollections(path.Join(folder, ".placeholder"))
	entries, err := s.client.List(folder)
	if err != nil {
		return nil, fmt.Errorf("list nextcloud folder: %w", err)
	}

	pictures := make([]ProductPictureInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		if !isImageFile(entry) {
			continue
		}
		pictures = append(pictures, ProductPictureInfo{
			FileName:    path.Base(entry.Path),
			Size:        entry.Size,
			ContentType: entry.ContentType,
			ModifiedAt:  entry.ModTime,
		})
	}

	return pictures, nil
}

// DownloadPicture streams a picture for a product by filename.
func (s *ProductPictureService) DownloadPicture(productName, fileName string) (io.ReadCloser, string, error) {
	if !s.Enabled() {
		return nil, "", fmt.Errorf("product pictures not configured")
	}
	if fileName == "" {
		return nil, "", fmt.Errorf("filename required")
	}
	safeName := sanitizeFileName(fileName)
	body, ct, err := s.client.Download(s.productFilePath(productName, safeName))
	if err != nil {
		return nil, "", err
	}
	return body, ct, nil
}

// DownloadPictureWithVariant returns the requested variant (thumb/preview) or the original image.
// Variants are cached as compressed WebP or JPEG on disk for faster subsequent loads.
// format can be "webp", "jpeg", or "" (defaults to webp for better compression).
func (s *ProductPictureService) DownloadPictureWithVariant(productName, fileName, variant, format string) (io.ReadCloser, string, error) {
	if variant == "" {
		return s.DownloadPicture(productName, fileName)
	}
	maxDim := 0
	normalizedVariant := strings.ToLower(variant)
	var variantDir string
	var legacyDir string // legacy cache dir for alias variants (backward compat)
	switch normalizedVariant {
	case "thumb", "thumbnail":
		maxDim = 480
		variantDir = "thumb"
		if normalizedVariant == "thumbnail" {
			legacyDir = "thumbnail"
		}
	case "preview", "medium":
		maxDim = 1200
		variantDir = "preview"
		if normalizedVariant == "medium" {
			legacyDir = "medium"
		}
	default:
		return s.DownloadPicture(productName, fileName)
	}

	if maxDim == 0 {
		return s.DownloadPicture(productName, fileName)
	}

	// Default format is webp for better compression
	if format == "" {
		format = "webp"
	}
	format = strings.ToLower(format)

	var fileExt, contentType string
	switch format {
	case "webp":
		fileExt = ".webp"
		contentType = "image/webp"
	case "jpeg", "jpg":
		fileExt = ".jpg"
		contentType = "image/jpeg"
	default:
		fileExt = ".webp"
		contentType = "image/webp"
	}

	safeFile := sanitizeFileName(fileName)
	cachePath := filepath.Join(s.cacheDir, variantDir, sanitizeFolderName(productName), safeFile+fileExt)
	if cached, err := os.Open(cachePath); err == nil {
		return cached, contentType, nil
	}
	// Also check legacy cache directories for alias variants to preserve backward compatibility.
	if legacyDir != "" {
		legacyCachePath := filepath.Join(s.cacheDir, legacyDir, sanitizeFolderName(productName), safeFile+fileExt)
		if cached, err := os.Open(legacyCachePath); err == nil {
			return cached, contentType, nil
		}
	}

	orig, origContentType, err := s.DownloadPicture(productName, safeFile)
	if err != nil {
		return nil, "", err
	}
	defer orig.Close()

	origBytes, err := io.ReadAll(orig)
	if err != nil {
		return nil, "", err
	}

	img, err := imaging.Decode(bytes.NewReader(origBytes), imaging.AutoOrientation(true))
	if err != nil {
		// If we cannot decode (e.g. HEIC unsupported), return original stream.
		return io.NopCloser(bytes.NewReader(origBytes)), origContentType, nil
	}

	resized := imaging.Fit(img, maxDim, maxDim, imaging.Lanczos)
	buf := bytes.Buffer{}

	// Encode based on requested format
	switch format {
	case "webp":
		// WebP encoding with quality 85 (25-35% smaller than JPEG)
		if err := webp.Encode(&buf, resized, &webp.Options{Quality: 85}); err != nil {
			// Fallback to JPEG if WebP encoding fails
			if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
				return io.NopCloser(bytes.NewReader(origBytes)), origContentType, nil
			}
			fileExt = ".jpg"
			contentType = "image/jpeg"
		}
	case "jpeg", "jpg":
		if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
			return io.NopCloser(bytes.NewReader(origBytes)), origContentType, nil
		}
	}

	cachePath = filepath.Join(s.cacheDir, variantDir, sanitizeFolderName(productName), safeFile+fileExt)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
		_ = os.WriteFile(cachePath, buf.Bytes(), 0o644)
	}

	return io.NopCloser(bytes.NewReader(buf.Bytes())), contentType, nil
}

// WarmPictureVariants generates cached variants in the background to improve perceived speed.
// Uses WebP format by default for optimal compression.
func (s *ProductPictureService) WarmPictureVariants(productName, fileName string) {
	if s == nil || !s.Enabled() {
		return
	}
	go func() {
		// Warm up WebP variants (default format)
		if r, _, err := s.DownloadPictureWithVariant(productName, fileName, "thumb", "webp"); err != nil {
			log.Printf("[PICTURES] Warm thumb (webp) failed for %s: %v", fileName, err)
		} else if r != nil {
			r.Close()
		}
		if r, _, err := s.DownloadPictureWithVariant(productName, fileName, "preview", "webp"); err != nil {
			log.Printf("[PICTURES] Warm preview (webp) failed for %s: %v", fileName, err)
		} else if r != nil {
			r.Close()
		}
		// Also warm up JPEG variants as fallback for old browsers
		if r, _, err := s.DownloadPictureWithVariant(productName, fileName, "thumb", "jpeg"); err != nil {
			log.Printf("[PICTURES] Warm thumb (jpeg) failed for %s: %v", fileName, err)
		} else if r != nil {
			r.Close()
		}
		if r, _, err := s.DownloadPictureWithVariant(productName, fileName, "preview", "jpeg"); err != nil {
			log.Printf("[PICTURES] Warm preview (jpeg) failed for %s: %v", fileName, err)
		} else if r != nil {
			r.Close()
		}
	}()
}

// ClearCachedVariants removes cached thumbnails/previews for a given file.
func (s *ProductPictureService) ClearCachedVariants(productName, fileName string) {
	if s == nil || s.cacheDir == "" {
		return
	}
	variants := []string{"thumb", "preview"}
	for _, variant := range variants {
		cachePath := filepath.Join(s.cacheDir, variant, sanitizeFolderName(productName), sanitizeFileName(fileName)+".jpg")
		_ = os.Remove(cachePath)
	}
}

// DeletePicture removes a specific picture from Nextcloud.
func (s *ProductPictureService) DeletePicture(productName, fileName string) error {
	if !s.Enabled() {
		return fmt.Errorf("product pictures not configured")
	}
	if fileName == "" {
		return fmt.Errorf("filename required")
	}
	return s.client.Delete(s.productFilePath(productName, sanitizeFileName(fileName)))
}

func (s *ProductPictureService) productFolder(productName string) string {
	return path.Join(s.rootFolder, sanitizeFolderName(productName))
}

func (s *ProductPictureService) productFilePath(productName, fileName string) string {
	return path.Join(s.productFolder(productName), sanitizeFileName(fileName))
}

func sanitizeFolderName(name string) string {
	clean := strings.TrimSpace(name)
	clean = strings.ReplaceAll(clean, "..", "")
	clean = strings.ReplaceAll(clean, "/", "-")
	clean = strings.ReplaceAll(clean, "\\", "-")
	clean = strings.ReplaceAll(clean, ":", "-")
	if clean == "" {
		return "unbenanntes-produkt"
	}
	return clean
}

func sanitizeFileName(name string) string {
	name = path.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	if name == "" {
		return "bild"
	}
	return name
}

func trimAndUnquote(val string) string {
	val = strings.TrimSpace(val)
	val = strings.Trim(val, "\"")
	return val
}

func isImageFile(entry storage.RemoteEntry) bool {
	if strings.HasPrefix(strings.ToLower(entry.ContentType), "image/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(entry.Path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif":
		return true
	default:
		return false
	}
}
