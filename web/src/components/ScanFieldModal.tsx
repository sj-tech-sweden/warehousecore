import { useState, useCallback, useEffect } from 'react';
import { X, Camera, Nfc, Keyboard } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ModalPortal } from './ModalPortal';
import { useBarcodeScanner } from '../hooks/useBarcodeScanner';
import { useNFCScanner } from '../hooks/useNFCScanner';
import { useBlockBodyScroll } from '../hooks/useBlockBodyScroll';

type InputMethod = 'keyboard' | 'camera' | 'nfc';

interface ScanFieldModalProps {
  isOpen: boolean;
  fieldLabel: string;
  onConfirm: (value: string) => void;
  onClose: () => void;
}

export function ScanFieldModal({ isOpen, fieldLabel, onConfirm, onClose }: ScanFieldModalProps) {
  const { t } = useTranslation();
  const [inputMethod, setInputMethod] = useState<InputMethod>('keyboard');
  const [manualValue, setManualValue] = useState('');
  const [scannedValue, setScannedValue] = useState('');

  useBlockBodyScroll(isOpen);

  const handleCodeDetected = useCallback((code: string) => {
    setScannedValue(code);
  }, []);

  const barcodeScanner = useBarcodeScanner({ onDetected: handleCodeDetected });
  const nfcScanner = useNFCScanner({ onDetected: handleCodeDetected });

  // Start/stop scanners when input method changes
  useEffect(() => {
    if (inputMethod !== 'camera') barcodeScanner.stopScanning();
    if (inputMethod !== 'nfc') nfcScanner.stopScanning();

    if (inputMethod === 'camera') {
      barcodeScanner.startScanning();
    } else if (inputMethod === 'nfc') {
      nfcScanner.startScanning();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [inputMethod]);

  // Stop scanners when modal closes
  useEffect(() => {
    if (!isOpen) {
      barcodeScanner.stopScanning();
      nfcScanner.stopScanning();
      setScannedValue('');
      setManualValue('');
      setInputMethod('keyboard');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      barcodeScanner.stopScanning();
      nfcScanner.stopScanning();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleInputMethodChange = useCallback((method: InputMethod) => {
    setScannedValue('');
    setInputMethod(method);
  }, []);

  const selectedValue = inputMethod === 'keyboard' ? manualValue : scannedValue;
  const trimmedSelectedValue = selectedValue.trim();

  const handleConfirm = () => {
    if (!trimmedSelectedValue) return;
    onConfirm(trimmedSelectedValue);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <ModalPortal>
      <div className="fixed inset-0 z-[200] flex items-center justify-center p-4">
        <div className="absolute inset-0 bg-black/70" onClick={onClose} />
        <div className="relative bg-dark-card border border-white/10 rounded-2xl w-full max-w-md shadow-2xl">
          {/* Header */}
          <div className="flex items-center justify-between p-4 border-b border-white/10">
            <h2 className="text-lg font-semibold text-white">
              {t('scanField.title', { field: fieldLabel })}
            </h2>
            <button type="button" onClick={onClose} aria-label={t('common.close')} title={t('common.close')} className="text-gray-400 hover:text-white transition-colors">
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="p-4 space-y-4">
            {/* Input method selector */}
            <div className="flex gap-1 p-1 bg-white/5 rounded-xl">
              <button
                type="button"
                onClick={() => handleInputMethodChange('keyboard')}
                className={`flex-1 flex items-center justify-center gap-1.5 py-2 rounded-lg text-xs sm:text-sm font-semibold transition-all ${
                  inputMethod === 'keyboard'
                    ? 'bg-accent-red text-white'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                <Keyboard className="w-4 h-4" />
                {t('scan.inputMethods.keyboard')}
              </button>
              {barcodeScanner.isSupported && (
                <button
                  type="button"
                  onClick={() => handleInputMethodChange('camera')}
                  className={`flex-1 flex items-center justify-center gap-1.5 py-2 rounded-lg text-xs sm:text-sm font-semibold transition-all ${
                    inputMethod === 'camera'
                      ? 'bg-accent-red text-white'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  <Camera className="w-4 h-4" />
                  {t('scan.inputMethods.camera')}
                </button>
              )}
              {nfcScanner.isSupported && (
                <button
                  type="button"
                  onClick={() => handleInputMethodChange('nfc')}
                  className={`flex-1 flex items-center justify-center gap-1.5 py-2 rounded-lg text-xs sm:text-sm font-semibold transition-all ${
                    inputMethod === 'nfc'
                      ? 'bg-accent-red text-white'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  <Nfc className="w-4 h-4" />
                  {t('scan.inputMethods.nfc')}
                </button>
              )}
            </div>

            {/* Camera Preview */}
            {inputMethod === 'camera' && (
              <div>
                {barcodeScanner.error ? (
                  <div className="flex items-center gap-3 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
                    <X className="w-5 h-5 flex-shrink-0" />
                    {t(barcodeScanner.error)}
                  </div>
                ) : (
                  <div className="relative rounded-xl overflow-hidden bg-black aspect-video">
                    <video
                      ref={barcodeScanner.videoRef}
                      className="w-full h-full object-cover"
                      playsInline
                      muted
                    />
                    {!barcodeScanner.isScanning && (
                      <div className="absolute inset-0 flex items-center justify-center bg-black/60">
                        <p className="text-white text-sm">{t('scan.camera.starting')}</p>
                      </div>
                    )}
                    {/* Scan overlay */}
                    <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                      <div className="w-2/3 h-2/3 border-2 border-accent-red/70 rounded-lg" />
                    </div>
                  </div>
                )}
                <p className="text-center text-gray-400 text-xs mt-2">{t('scan.camera.hint')}</p>
              </div>
            )}

            {/* NFC Waiting State */}
            {inputMethod === 'nfc' && (
              <div>
                {nfcScanner.error ? (
                  <div className="flex items-center gap-3 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
                    <X className="w-5 h-5 flex-shrink-0" />
                    {t(nfcScanner.error)}
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center gap-3 p-6 rounded-xl bg-white/5 border border-white/10">
                    <div className={`p-4 rounded-full ${nfcScanner.isScanning ? 'bg-accent-red/20 animate-pulse' : 'bg-white/10'}`}>
                      <Nfc className={`w-12 h-12 ${nfcScanner.isScanning ? 'text-accent-red' : 'text-gray-500'}`} />
                    </div>
                    <p className="text-white text-sm font-semibold">
                      {nfcScanner.isScanning ? t('scan.nfc.ready') : t('scan.nfc.starting')}
                    </p>
                    <p className="text-gray-400 text-xs text-center">{t('scan.nfc.hint')}</p>
                  </div>
                )}
              </div>
            )}

            {/* Keyboard / Manual input */}
            {inputMethod === 'keyboard' && (
              <input
                type="text"
                value={manualValue}
                onChange={(e) => setManualValue(e.target.value)}
                placeholder={t('scanField.placeholder', { field: fieldLabel })}
                className="input-field w-full"
                autoFocus
              />
            )}

            {/* Scanned value preview (camera/nfc) */}
            {inputMethod !== 'keyboard' && scannedValue && (
              <div className="p-3 rounded-xl bg-green-500/10 border border-green-500/30 text-green-400 text-sm font-mono break-all">
                {scannedValue}
              </div>
            )}

            {/* Actions */}
            <div className="flex gap-3 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="flex-1 px-4 py-2 rounded-xl border border-white/20 text-gray-300 hover:text-white hover:border-white/40 transition-colors text-sm"
              >
                {t('common.cancel')}
              </button>
              <button
                type="button"
                onClick={handleConfirm}
                disabled={trimmedSelectedValue.length === 0}
                className="flex-1 px-4 py-2 rounded-xl bg-accent-red text-white font-semibold text-sm hover:bg-accent-red/80 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t('scanField.apply')}
              </button>
            </div>
          </div>
        </div>
      </div>
    </ModalPortal>
  );
}
