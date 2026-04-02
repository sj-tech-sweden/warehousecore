import { useState, useRef, useCallback, useEffect } from 'react';

interface NFCScannerOptions {
  onDetected: (code: string) => void;
}

interface NFCScannerResult {
  isSupported: boolean;
  isScanning: boolean;
  startScanning: () => Promise<void>;
  stopScanning: () => void;
  error: string | null;
}

export function useNFCScanner({ onDetected }: NFCScannerOptions): NFCScannerResult {
  const [isScanning, setIsScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const readerRef = useRef<NDEFReader | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  const isSupported = typeof window !== 'undefined' && 'NDEFReader' in window;

  const stopScanning = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    readerRef.current = null;
    setIsScanning(false);
  }, []);

  const startScanning = useCallback(async () => {
    setError(null);

    if (!isSupported) {
      setError('scan.nfc.errors.notSupported');
      return;
    }

    try {
      const reader = new NDEFReader();
      readerRef.current = reader;

      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      reader.addEventListener('reading', (event: NDEFReadingEvent) => {
        for (const record of event.message.records) {
          let text: string | null = null;

          if (record.recordType === 'text' && record.data) {
            const decoder = new TextDecoder(record.encoding ?? 'utf-8');
            text = decoder.decode(record.data);
          } else if (record.recordType === 'url' && record.data) {
            const decoder = new TextDecoder('utf-8');
            text = decoder.decode(record.data);
          } else if (record.id) {
            text = record.id;
          }

          if (text) {
            onDetected(text);
            return;
          }
        }

        // Fall back to the tag serial number if no readable record
        if (event.serialNumber) {
          onDetected(event.serialNumber);
        }
      });

      reader.addEventListener('readingerror', () => {
        setError('scan.nfc.errors.readError');
      });

      await reader.scan({ signal: abortController.signal });
      setIsScanning(true);
    } catch (err: unknown) {
      if ((err as { name?: string })?.name === 'AbortError') return;
      setError('scan.nfc.errors.startFailed');
      setIsScanning(false);
    }
  }, [isSupported, onDetected]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      stopScanning();
    };
  }, [stopScanning]);

  return { isSupported, isScanning, startScanning, stopScanning, error };
}
