package led

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"warehousecore/internal/models"
)

// Publisher handles MQTT publishing for LED commands
type Publisher struct {
	client    mqtt.Client
	config    PublisherConfig
	connected bool
	mu        sync.RWMutex
	dryRun    bool
}

// PublisherConfig holds MQTT connection configuration
type PublisherConfig struct {
	Host        string
	Port        int
	UseTLS      bool
	Username    string
	Password    string
	TopicPrefix string
	WarehouseID string
	ClientID    string
}

var (
	publisherInstance *Publisher
	publisherOnce     sync.Once
)

// GetPublisher returns the singleton MQTT publisher instance
func GetPublisher() *Publisher {
	publisherOnce.Do(func() {
		publisherInstance = NewPublisher()
	})
	return publisherInstance
}

// NewPublisher creates a new MQTT publisher from environment variables
func NewPublisher() *Publisher {
	config := PublisherConfig{
		Host:        os.Getenv("LED_MQTT_HOST"),
		Port:        getEnvInt("LED_MQTT_PORT", 1883),
		UseTLS:      getEnvBool("LED_MQTT_TLS", false),
		Username:    os.Getenv("LED_MQTT_USER"),
		Password:    os.Getenv("LED_MQTT_PASS"),
		TopicPrefix: getEnvString("LED_MQTT_TOPIC_PREFIX", "weidelbach"),
		WarehouseID: getEnvString("LED_WAREHOUSE_ID", "WDL"),
		ClientID:    fmt.Sprintf("storagecore-%d", time.Now().Unix()),
	}

	// Check if MQTT is configured (dry-run mode if not)
	dryRun := config.Host == ""
	if dryRun {
		log.Println("[LED] MQTT not configured - running in DRY-RUN mode (commands will be logged only)")
		return &Publisher{
			config:    config,
			connected: false,
			dryRun:    true,
		}
	}

	pub := &Publisher{
		config: config,
		dryRun: false,
	}

	retries := getEnvInt("LED_MQTT_CONNECT_RETRIES", 10)
	if retries < 0 {
		retries = 0
	}
	delayMS := getEnvInt("LED_MQTT_CONNECT_RETRY_DELAY_MS", 2000)
	if delayMS < 100 {
		delayMS = 100
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(delayMS) * time.Millisecond)
		}
		if err := pub.connect(); err != nil {
			lastErr = err
			log.Printf("[LED] MQTT connect attempt %d/%d failed: %v", attempt+1, retries+1, err)
			continue
		}
		lastErr = nil
		break
	}

	if lastErr != nil {
		log.Printf("[LED] Failed to connect to MQTT broker after %d attempts: %v - falling back to DRY-RUN mode", retries+1, lastErr)
		pub.dryRun = true
	}

	return pub
}

// connect establishes connection to MQTT broker
func (p *Publisher) connect() error {
	opts := mqtt.NewClientOptions()

	// Build broker URL
	scheme := "tcp"
	if p.config.UseTLS {
		scheme = "ssl"
		// Configure TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
		}
		opts.SetTLSConfig(tlsConfig)
	}

	brokerURL := fmt.Sprintf("%s://%s:%d", scheme, p.config.Host, p.config.Port)
	opts.AddBroker(brokerURL)
	opts.SetClientID(p.config.ClientID)

	if p.config.Username != "" {
		opts.SetUsername(p.config.Username)
		opts.SetPassword(p.config.Password)
	}

	// Set connection callbacks
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		p.mu.Lock()
		p.connected = false
		p.mu.Unlock()
		log.Printf("[LED] MQTT connection lost: %v", err)
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		p.mu.Lock()
		p.connected = true
		p.mu.Unlock()
		log.Printf("[LED] MQTT connected to %s", brokerURL)
	})

	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	// Create and connect client
	p.client = mqtt.NewClient(opts)

	token := p.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect: %w", token.Error())
	}

	return nil
}

// IsConnected returns true if the MQTT client is connected
func (p *Publisher) IsConnected() bool {
	if p.dryRun {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected && p.client.IsConnected()
}

// IsDryRun returns true if publisher is in dry-run mode
func (p *Publisher) IsDryRun() bool {
	return p.dryRun
}

// PublishCommand publishes an LED command to the MQTT topic
func (p *Publisher) PublishCommand(cmd LEDCommand) error {
	// Ensure warehouse_id is set
	if cmd.WarehouseID == "" {
		cmd.WarehouseID = p.config.WarehouseID
	}

	// Serialize command to JSON (compact to avoid exceeding firmware buffer)
	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}
	if len(payload) > 3800 {
		log.Printf("[LED] Warning: command payload size %d bytes approaches firmware limit", len(payload))
	}

	// Build topic: <prefix>/<warehouse_id>/cmd
	topic := fmt.Sprintf("%s/%s/cmd", p.config.TopicPrefix, cmd.WarehouseID)

	// Dry-run mode: just log
	if p.dryRun {
		log.Printf("[LED] DRY-RUN: Would publish to topic '%s': %s", topic, string(payload))
		return nil
	}

	// Check connection
	if !p.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	if cmd.Op == "highlight" {
		log.Printf("[LED] MQTT payload: %s", string(payload))
	}

	// Publish with QoS 1 (at least once delivery)
	token := p.client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish: %w", token.Error())
	}

	log.Printf("[LED] Published '%s' command to %s (%d bins)", cmd.Op, topic, countBins(cmd))
	return nil
}

// PublishHighlight sends a highlight command for the given job with the provided appearance settings
func (p *Publisher) PublishHighlight(jobID string, mapping *LEDMapping, deviceZones map[string]string, settings *models.LEDJobHighlightSettings) error {
	if mapping == nil {
		return fmt.Errorf("mapping configuration not loaded")
	}

	// Build command from device zones
	cmd := LEDCommand{
		Op:          "highlight",
		WarehouseID: mapping.WarehouseID,
		Shelves:     []Shelf{},
	}

	if settings == nil {
		settings = models.DefaultLEDJobHighlightSettings()
	} else {
		settings.Normalize(models.DefaultLEDJobHighlightSettings())
	}

	if settings.Mode == "required_only" {
		if err := p.PublishClear(); err != nil {
			log.Printf("[LED] Failed to clear LEDs before required-only highlight: %v", err)
		}
	}

	// Create a map of zones that have job devices (for quick lookup)
	jobZones := make(map[string]bool)
	for _, zoneName := range deviceZones {
		jobZones[zoneName] = true
	}

	// Group bins by shelf - illuminate ALL bins in the mapping
	shelfBins := make(map[string][]Bin)

	// Iterate through ALL bins in mapping
	for _, shelf := range mapping.Shelves {
		for _, binConfig := range shelf.Bins {
			// Check if this bin has a job device
			hasJobDevice := jobZones[binConfig.BinID]

			appearance := settings.NonRequired
			if hasJobDevice {
				appearance = settings.Required
			} else if settings.Mode == "required_only" {
				// Skip bins without pending devices when operating in required-only mode
				continue
			}

			// Create bin entry
			bin := Bin{
				BinID:     binConfig.BinID,
				Pixels:    binConfig.Pixels,
				Color:     appearance.Color,
				Pattern:   appearance.Pattern,
				Intensity: int(appearance.Intensity),
			}
			if appearance.Speed > 0 {
				bin.Speed = appearance.Speed
			}

			shelfBins[shelf.ShelfID] = append(shelfBins[shelf.ShelfID], bin)
		}
	}

	// Convert map to shelves array
	for shelfID, bins := range shelfBins {
		cmd.Shelves = append(cmd.Shelves, Shelf{
			ShelfID: shelfID,
			Bins:    bins,
		})
	}

	if len(cmd.Shelves) == 0 {
		return fmt.Errorf("no bins configured in LED mapping")
	}

	if settings.Mode == "required_only" {
		log.Printf("[LED] Highlighting %d required bins only", len(jobZones))
	} else {
		log.Printf("[LED] Highlighting %d bins total (%d required, %d non-required)",
			countTotalBins(shelfBins),
			len(jobZones),
			countTotalBins(shelfBins)-len(jobZones))
	}

	return p.PublishCommand(cmd)
}

// countTotalBins counts total bins in the shelfBins map
func countTotalBins(shelfBins map[string][]Bin) int {
	count := 0
	for _, bins := range shelfBins {
		count += len(bins)
	}
	return count
}

// PublishClear sends a clear command to turn off all LEDs
func (p *Publisher) PublishClear() error {
	cmd := LEDCommand{
		Op:          "clear",
		WarehouseID: p.config.WarehouseID,
	}
	return p.PublishCommand(cmd)
}

// PublishIdentify sends an identify command (blink all LEDs for testing)
func (p *Publisher) PublishIdentify() error {
	cmd := LEDCommand{
		Op:          "identify",
		WarehouseID: p.config.WarehouseID,
	}
	return p.PublishCommand(cmd)
}

// Close disconnects the MQTT client
func (p *Publisher) Close() {
	if p.client != nil && p.client.IsConnected() {
		p.client.Disconnect(250)
		log.Println("[LED] MQTT client disconnected")
	}
}

// Helper functions

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func countBins(cmd LEDCommand) int {
	count := 0
	for _, shelf := range cmd.Shelves {
		count += len(shelf.Bins)
	}
	return count
}

// findBinPixels searches for bin in mapping and returns pixel indices
func findBinPixels(mapping *LEDMapping, zoneName string) ([]int, bool) {
	for _, shelf := range mapping.Shelves {
		for _, bin := range shelf.Bins {
			if bin.BinID == zoneName {
				return bin.Pixels, true
			}
		}
	}
	return nil, false
}

// parseZoneName extracts shelf ID and bin ID from zone name
// Examples: "A-01" -> ("A", "A-01"), "B-03" -> ("B", "B-03")
func parseZoneName(zoneName string) (shelfID, binID string) {
	if len(zoneName) < 1 {
		return "", ""
	}

	// Simple heuristic: first character is shelf ID
	// For more complex naming, this can be enhanced
	shelfID = string(zoneName[0])
	binID = zoneName

	return shelfID, binID
}
