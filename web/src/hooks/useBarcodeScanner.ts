import { useRef, useState, useCallback, useEffect } from 'react';

interface BarcodeScannerOptions {
  onDetected: (code: string) => void;
}

interface BarcodeScannerResult {
  isSupported: boolean;
  isScanning: boolean;
  startScanning: () => Promise<void>;
  stopScanning: () => void;
  videoRef: React.RefObject<HTMLVideoElement | null>;
  error: string | null;
}


export function useBarcodeScanner({ onDetected }: BarcodeScannerOptions): BarcodeScannerResult {
  const [isScanning, setIsScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const animationFrameRef = useRef<number | null>(null);
  const detectorRef = useRef<BarcodeDetector | null>(null);
  const lastDetectedRef = useRef<string | null>(null);
  const lastDetectedTimeRef = useRef<number>(0);

  const isSupported = typeof window !== 'undefined' && 'BarcodeDetector' in window;

  const stopScanning = useCallback(() => {
    if (animationFrameRef.current !== null) {
      cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }
    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
    setIsScanning(false);
  }, []);

  const startScanning = useCallback(async () => {
    setError(null);

    if (!isSupported) {
      setError('scan.camera.errors.notSupported');
      return;
    }

    try {
      detectorRef.current = new window.BarcodeDetector({
        formats: [
          'aztec', 'code_128', 'code_39', 'code_93', 'codabar',
          'data_matrix', 'ean_13', 'ean_8', 'itf', 'pdf417',
          'qr_code', 'upc_a', 'upc_e',
        ],
      });
    } catch {
      setError('scan.camera.errors.initFailed');
      return;
    }

    let stream: MediaStream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment', width: { ideal: 1280 }, height: { ideal: 720 } },
      });
    } catch {
      setError('scan.camera.errors.accessDenied');
      return;
    }

    streamRef.current = stream;

    if (videoRef.current) {
      videoRef.current.srcObject = stream;
      try {
        await videoRef.current.play();
      } catch {
        setError('scan.camera.errors.playFailed');
        stopScanning();
        return;
      }
    }

    setIsScanning(true);
    lastDetectedRef.current = null;
    lastDetectedTimeRef.current = 0;

    const detect = async () => {
      if (!videoRef.current || !detectorRef.current) return;
      const video = videoRef.current;

      if (video.readyState === video.HAVE_ENOUGH_DATA) {
        try {
          const barcodes = await detectorRef.current.detect(video);
          if (barcodes.length > 0) {
            const code = barcodes[0].rawValue;
            const now = Date.now();
            // Debounce: same code must not re-fire within 2 seconds
            if (code !== lastDetectedRef.current || now - lastDetectedTimeRef.current > 2000) {
              lastDetectedRef.current = code;
              lastDetectedTimeRef.current = now;
              onDetected(code);
            }
          }
        } catch {
          // Detection errors are expected on frames without barcodes
        }
      }

      animationFrameRef.current = requestAnimationFrame(detect);
    };

    animationFrameRef.current = requestAnimationFrame(detect);
  }, [isSupported, onDetected, stopScanning]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      stopScanning();
    };
  }, [stopScanning]);

  return { isSupported, isScanning, startScanning, stopScanning, videoRef, error };
}
