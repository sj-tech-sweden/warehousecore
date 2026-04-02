// Type declarations for experimental browser APIs not yet in TypeScript's standard DOM lib

// BarcodeDetector API
// https://developer.mozilla.org/en-US/docs/Web/API/BarcodeDetector
type BarcodeFormat =
  | 'aztec'
  | 'code_128'
  | 'code_39'
  | 'code_93'
  | 'codabar'
  | 'data_matrix'
  | 'ean_13'
  | 'ean_8'
  | 'itf'
  | 'pdf417'
  | 'qr_code'
  | 'upc_a'
  | 'upc_e'
  | 'unknown';

interface BarcodeDetectorOptions {
  formats?: BarcodeFormat[];
}

interface DetectedBarcode {
  boundingBox: DOMRectReadOnly;
  cornerPoints: ReadonlyArray<{ x: number; y: number }>;
  format: BarcodeFormat;
  rawValue: string;
}

declare class BarcodeDetector {
  constructor(options?: BarcodeDetectorOptions);
  static getSupportedFormats(): Promise<BarcodeFormat[]>;
  detect(image: ImageBitmapSource | HTMLVideoElement): Promise<DetectedBarcode[]>;
}

// Web NFC API
// https://developer.mozilla.org/en-US/docs/Web/API/Web_NFC_API
interface NDEFRecord {
  recordType: string;
  mediaType?: string;
  id?: string;
  data?: DataView;
  encoding?: string;
  lang?: string;
}

interface NDEFMessage {
  records: NDEFRecord[];
}

interface NDEFReadingEvent extends Event {
  serialNumber: string;
  message: NDEFMessage;
}

interface NDEFReader extends EventTarget {
  scan(options?: { signal?: AbortSignal }): Promise<void>;
  addEventListener(type: 'reading', listener: (event: NDEFReadingEvent) => void): void;
  addEventListener(type: 'readingerror', listener: (event: Event) => void): void;
  addEventListener(type: string, listener: EventListenerOrEventListenerObject): void;
}

declare const NDEFReader: {
  prototype: NDEFReader;
  new (): NDEFReader;
};

// Augment Window with both APIs
declare interface Window {
  BarcodeDetector: typeof BarcodeDetector;
  NDEFReader: typeof NDEFReader;
}

