import { useState, useCallback, useRef, useEffect } from 'react';
import { ScanLine, CheckCircle, XCircle, MapPin, Lightbulb, Wrench, AlertTriangle, Info, Camera, Nfc, Keyboard, X, Box, Trash2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { scansApi, zonesApi, jobsApi, ledApi, maintenanceApi, casesApi } from '../lib/api';
import type { Device, ScanResponse, CaseDetail } from '../lib/api';
import { useBlockBodyScroll } from '../hooks/useBlockBodyScroll';
import { DeviceInfoModal } from '../components/DeviceInfoModal';
import { useBarcodeScanner } from '../hooks/useBarcodeScanner';
import { useNFCScanner } from '../hooks/useNFCScanner';

import type { InputMethod } from '../types/scanTypes';
type ScanStep = 'device' | 'zone' | 'case' | 'device-for-case';

export function ScanPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [scanCode, setScanCode] = useState('');
  const [action, setAction] = useState<'intake' | 'outtake' | 'check' | 'case'>('check');
  const [result, setResult] = useState<ScanResponse | null>(null);
  const [loading, setLoading] = useState(false);

  // Input method: keyboard (default), camera, or nfc
  const [inputMethod, setInputMethod] = useState<InputMethod>('keyboard');

  // Two-step workflow for intake
  const [step, setStep] = useState<ScanStep>('device');
  const [deviceScanCode, setDeviceScanCode] = useState('');
  const [consumableQuantity, setConsumableQuantity] = useState<number | undefined>(undefined);

  // Case scanning workflow
  const [scannedCase, setScannedCase] = useState<CaseDetail | null>(null);
  const [caseDeviceIds, setCaseDeviceIds] = useState<string[]>([]);
  const [caseActionMessage, setCaseActionMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Job-Code scan states
  const [showLEDModal, setShowLEDModal] = useState(false);
  const [scannedJobId, setScannedJobId] = useState<number | null>(null);

  // Device detail modal state
  const [viewDevice, setViewDevice] = useState<Device | null>(null);

  // Service modal states
  const [showServiceModal, setShowServiceModal] = useState(false);
  const [serviceForm, setServiceForm] = useState({
    device_id: '',
    severity: 'medium',
    title: '',
    description: '',
  });
  const [serviceLoading, setServiceLoading] = useState(false);
  const [serviceSuccess, setServiceSuccess] = useState(false);
  const [serviceError, setServiceError] = useState<string | null>(null);

  // Block body scroll when LED modal or service modal is open
  useBlockBodyScroll(showLEDModal || showServiceModal);

  // Keep a stable ref to the current scan submission handler so the scanner
  // callbacks (which are memoised) can always call the latest version.
  const submitCodeRef = useRef<(code: string) => void>(() => {});

  // Ref for case action message auto-dismiss timeout – cleared on unmount to
  // prevent state updates on an unmounted component.
  const caseActionTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const scheduleCaseActionDismiss = useCallback((ms: number) => {
    if (caseActionTimeoutRef.current !== null) {
      clearTimeout(caseActionTimeoutRef.current);
    }
    caseActionTimeoutRef.current = setTimeout(() => {
      setCaseActionMessage(null);
      caseActionTimeoutRef.current = null;
    }, ms);
  }, []);

  const handleCodeDetected = useCallback((code: string) => {
    submitCodeRef.current(code);
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
    // Intentionally not including scanner methods in deps to avoid loops
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [inputMethod]);

  // Stop scanners on unmount; also clear any pending case-action-message timeout
  useEffect(() => {
    return () => {
      barcodeScanner.stopScanning();
      nfcScanner.stopScanning();
      if (caseActionTimeoutRef.current !== null) {
        clearTimeout(caseActionTimeoutRef.current);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleJobCodeScan = useCallback(async (jobId: number) => {
    setScanCode('');
    setLoading(true);

    try {
      // First, verify job exists
      await jobsApi.getById(jobId);

      // Check LED status
      const { data: ledStatus } = await ledApi.getStatus();

      if (ledStatus.mqtt_connected) {
        // LED is on - navigate directly to job
        await ledApi.highlightJob(jobId);
        navigate(`/jobs/${jobId}`);
      } else {
        // LED is off - ask user if they want to enable it
        setScannedJobId(jobId);
        setShowLEDModal(true);
      }
    } catch (error: any) {
      console.error('Job scan failed:', error);
      setResult({
        success: false,
        message: error.response?.data?.error || t('jobsPage.jobNotFound', { id: jobId }),
        action: 'check',
        duplicate: false,
      });
    } finally {
      setLoading(false);
    }
  }, [navigate, t]);

  const processCode = useCallback(async (code: string) => {
    if (!code.trim()) return;

    // Check if scan code is a Job-Code (format: JOB######)
    const jobCodeMatch = code.match(/^JOB(\d{6})$/i);
    if (jobCodeMatch) {
      const jobId = parseInt(jobCodeMatch[1], 10);
      await handleJobCodeScan(jobId);
      return;
    }

    setLoading(true);
    try {
      // Case scanning workflow: Step 1 - Scan case
      if (action === 'case' && step === 'case') {
        const { data: caseData } = await casesApi.getByScan(code);
        setScannedCase(caseData);
        setCaseDeviceIds([]);
        setCaseActionMessage(null);
        setStep('device-for-case');
        setScanCode('');
        setResult(null);
      }
      // Case scanning workflow: Step 2 - Scan devices to add
      else if (action === 'case' && step === 'device-for-case') {
        if (!scannedCase) return;

        // Verify device exists
        const { data: checkData } = await scansApi.process({
          scan_code: code,
          action: 'check',
        });

        if (!checkData.success || !checkData.device) {
          setCaseActionMessage({ type: 'error', text: checkData.message || t('scan.scanError') });
          setScanCode('');
          scheduleCaseActionDismiss(3000);
        } else {
          const deviceId = checkData.device.device_id;
          if (caseDeviceIds.includes(deviceId)) {
            setCaseActionMessage({ type: 'error', text: t('scan.case.alreadyScanned', { id: deviceId }) });
            setScanCode('');
            scheduleCaseActionDismiss(3000);
          } else {
            // Add device to case immediately
            const { data: addData } = await casesApi.addDevices(scannedCase.case_id, [deviceId]);
            if (addData.success_count > 0) {
              setCaseDeviceIds(prev => [...prev, deviceId]);
              setCaseActionMessage({ type: 'success', text: t('scan.case.deviceAdded', { id: deviceId }) });
            } else {
              const errMsg = addData.errors?.[0] ?? t('scan.case.addFailed');
              setCaseActionMessage({ type: 'error', text: errMsg });
            }
            setScanCode('');
            scheduleCaseActionDismiss(3000);
          }
        }
      }
      // Step 1: Scan device for intake
      else if (action === 'intake' && step === 'device') {
        // Verify device exists by trying to scan it (check action)
        const { data } = await scansApi.process({
          scan_code: code,
          action: 'check',
        });

        if (data.success) {
          // Check if this is an accessory/consumable (has product info with unit)
          if (data.product && data.product.unit) {
            // This is an accessory/consumable - ask for quantity
            const quantityStr = window.prompt(t('scan.prompts.intakeQuantity', { unit: data.product.unit }));

            if (!quantityStr || isNaN(Number(quantityStr)) || Number(quantityStr) <= 0) {
              setResult({
                success: false,
                message: t('scan.invalidQuantity'),
                action,
                duplicate: false,
              });
              setLoading(false);
              return;
            }

            // Store quantity and proceed to zone scan
            setConsumableQuantity(Number(quantityStr));
            setDeviceScanCode(code);
            setStep('zone');
            setScanCode('');
            setResult(null);
            setLoading(false);
            return;
          }

          // Regular device - proceed to zone scan
          setDeviceScanCode(code);
          setStep('zone');
          setScanCode('');
          setResult(null);
        } else {
          setResult(data);
        }
      }
      // Step 2: Scan zone for intake
      else if (action === 'intake' && step === 'zone') {
        // Find zone by barcode
        const { data: zone } = await zonesApi.getByScan(code);

        // Now process the actual intake with zone_id (and quantity if it's a consumable)
        const { data } = await scansApi.process({
          scan_code: deviceScanCode,
          action: 'intake',
          zone_id: zone.zone_id,
          job_id: consumableQuantity, // Pass quantity for consumables
        });

        setResult(data);
        setScanCode('');
        setDeviceScanCode('');
        setConsumableQuantity(undefined);
        setStep('device');
      }
      // All other actions (outtake, check) - single step
      else if (action !== 'case') {
        // For consumables with intake/outtake, ask for quantity first
        let quantity = undefined;
        if ((action === 'intake' || action === 'outtake')) {
          // First check if this might be a consumable (quick check without committing)
          const checkResponse = await scansApi.process({
            scan_code: code,
            action: 'check',
          });

          // If the response includes product info with a unit, it's an accessory/consumable
          if (checkResponse.data.product && checkResponse.data.product.unit) {
            const promptText = action === 'intake'
              ? t('scan.prompts.intakeQuantity', { unit: checkResponse.data.product.unit })
              : t('scan.prompts.outtakeQuantity', { unit: checkResponse.data.product.unit });
            const quantityStr = window.prompt(promptText);

            if (!quantityStr || isNaN(Number(quantityStr)) || Number(quantityStr) <= 0) {
              setResult({
                success: false,
                message: t('scan.invalidQuantity'),
                action,
                duplicate: false,
              });
              setLoading(false);
              return;
            }
            quantity = Number(quantityStr);
          }
        }

        // Now do the actual scan with quantity if provided
        const { data } = await scansApi.process({
          scan_code: code,
          action: action,
          job_id: quantity, // Pass quantity via job_id field (backend expects this)
        });
        setResult(data);
        setScanCode('');
      }
    } catch (error: any) {
      console.error('Scan failed:', error);

      if (action === 'case' && step === 'case') {
        setCaseActionMessage({ type: 'error', text: error.response?.data?.error || t('scan.case.caseNotFound') });
        setScanCode('');
        scheduleCaseActionDismiss(4000);
      } else if (action === 'case' && step === 'device-for-case') {
        setCaseActionMessage({ type: 'error', text: error.response?.data?.error || t('scan.scanError') });
        setScanCode('');
        scheduleCaseActionDismiss(3000);
      } else {
        setResult({
          success: false,
          message: error.response?.data?.error || t('scan.scanError'),
          action,
          duplicate: false,
        });

        // Reset to step 1 on error
        if (step === 'zone') {
          setStep('device');
          setDeviceScanCode('');
          setScanCode('');
        }
      }
    } finally {
      setLoading(false);
    }
  }, [action, step, deviceScanCode, consumableQuantity, scannedCase, caseDeviceIds, t, handleJobCodeScan, scheduleCaseActionDismiss]);

  // Keep submitCodeRef in sync with the latest processCode so scanner callbacks
  // (which are memoised on mount) can always reach the current state closure.
  useEffect(() => {
    submitCodeRef.current = processCode;
  }, [processCode]);

  const handleScan = (e: React.FormEvent) => {
    e.preventDefault();
    processCode(scanCode);
  };

  const handleActionChange = (newAction: 'intake' | 'outtake' | 'check' | 'case') => {
    setAction(newAction);
    if (newAction === 'case') {
      setStep('case');
      setScannedCase(null);
      setCaseDeviceIds([]);
      setCaseActionMessage(null);
    } else {
      setStep('device');
    }
    setDeviceScanCode('');
    setConsumableQuantity(undefined);
    setScanCode('');
    setResult(null);
  };

  const handleInputMethodChange = (method: InputMethod) => {
    setScanCode('');
    setResult(null);
    setInputMethod(method);
  };

  const handleLEDModalConfirm = async () => {
    if (!scannedJobId) return;

    try {
      setLoading(true);
      await ledApi.highlightJob(scannedJobId);
      setShowLEDModal(false);
      navigate(`/jobs/${scannedJobId}`);
    } catch (error) {
      console.error('LED activation failed:', error);
      setShowLEDModal(false);
      navigate(`/jobs/${scannedJobId}`);
    }
  };

  const handleLEDModalCancel = () => {
    if (scannedJobId) {
      navigate(`/jobs/${scannedJobId}`);
    }
    setShowLEDModal(false);
    setScannedJobId(null);
  };

  const openServiceModal = (deviceId: string, severity: 'medium' | 'high') => {
    setServiceForm({ device_id: deviceId, severity, title: '', description: '' });
    setServiceSuccess(false);
    setServiceError(null);
    setShowServiceModal(true);
  };

  const handleServiceSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setServiceLoading(true);
    setServiceError(null);
    try {
      await maintenanceApi.createDefect(serviceForm);
      setServiceSuccess(true);
    } catch (error: any) {
      console.error('Failed to create defect:', error);
      setServiceError(error.response?.data?.error || t('maintenance.createDefectError'));
    } finally {
      setServiceLoading(false);
    }
  };

  const handleServiceModalClose = () => {
    if (serviceLoading) return;
    setShowServiceModal(false);
    setServiceSuccess(false);
    setServiceError(null);
  };

  return (
    <div className="flex items-center justify-center p-3 sm:p-4">
      <div className="w-full max-w-2xl my-auto">
        {/* Scan Form */}
        <div className="glass-dark rounded-2xl sm:rounded-3xl p-4 sm:p-8 border-2 border-white/10">
          <div className="text-center mb-6 sm:mb-8">
            <div className="inline-block p-3 sm:p-4 rounded-xl sm:rounded-2xl bg-gradient-to-br from-accent-red to-red-700 mb-3 sm:mb-4">
              {step === 'zone' ? (
                <MapPin className="w-8 h-8 sm:w-12 sm:h-12 text-white" />
              ) : step === 'case' || step === 'device-for-case' ? (
                <Box className="w-8 h-8 sm:w-12 sm:h-12 text-white" />
              ) : (
                <ScanLine className="w-8 h-8 sm:w-12 sm:h-12 text-white" />
              )}
            </div>
            <h1 className="text-2xl sm:text-4xl font-bold text-white mb-1 sm:mb-2">
              {step === 'zone'
                ? t('scan.zoneTitle')
                : step === 'case'
                  ? t('scan.case.title')
                  : step === 'device-for-case'
                    ? t('scan.case.addDevicesTitle')
                    : t('scan.scannerTitle')}
            </h1>
            <p className="text-sm sm:text-base text-gray-400">
              {step === 'zone'
                ? t('scan.zoneSubtitle')
                : step === 'case'
                  ? t('scan.case.subtitle')
                  : step === 'device-for-case'
                    ? t('scan.case.addDevicesSubtitle')
                    : t('scan.scannerSubtitle')}
            </p>
          </div>

          {/* Step Indicator for Intake */}
          {action === 'intake' && (
            <div className="mb-4 sm:mb-6 flex items-center justify-center gap-2 sm:gap-4">
              <div className={`flex items-center gap-1.5 sm:gap-2 ${step === 'device' ? 'text-accent-red' : 'text-green-500'}`}>
                <div className={`w-7 h-7 sm:w-8 sm:h-8 rounded-full flex items-center justify-center text-sm sm:text-base ${
                  step === 'device' ? 'bg-accent-red' : 'bg-green-500'
                }`}>
                  {step === 'zone' ? '✓' : '1'}
                </div>
                <span className="text-sm sm:text-base font-semibold">{t('scan.steps.device')}</span>
              </div>
              <div className="w-8 sm:w-12 h-0.5 bg-white/20"></div>
              <div className={`flex items-center gap-1.5 sm:gap-2 ${step === 'zone' ? 'text-accent-red' : 'text-gray-500'}`}>
                <div className={`w-7 h-7 sm:w-8 sm:h-8 rounded-full flex items-center justify-center text-sm sm:text-base ${
                  step === 'zone' ? 'bg-accent-red' : 'bg-gray-700'
                }`}>
                  2
                </div>
                <span className="text-sm sm:text-base font-semibold">{t('scan.steps.zone')}</span>
              </div>
            </div>
          )}

          {/* Step Indicator for Case scanning */}
          {action === 'case' && (
            <div className="mb-4 sm:mb-6 flex items-center justify-center gap-2 sm:gap-4">
              <div className={`flex items-center gap-1.5 sm:gap-2 ${step === 'case' ? 'text-accent-red' : 'text-green-500'}`}>
                <div className={`w-7 h-7 sm:w-8 sm:h-8 rounded-full flex items-center justify-center text-sm sm:text-base ${
                  step === 'case' ? 'bg-accent-red' : 'bg-green-500'
                }`}>
                  {step !== 'case' ? '✓' : '1'}
                </div>
                <span className="text-sm sm:text-base font-semibold">{t('scan.case.steps.case')}</span>
              </div>
              <div className="w-8 sm:w-12 h-0.5 bg-white/20"></div>
              <div className={`flex items-center gap-1.5 sm:gap-2 ${step === 'device-for-case' ? 'text-accent-red' : 'text-gray-500'}`}>
                <div className={`w-7 h-7 sm:w-8 sm:h-8 rounded-full flex items-center justify-center text-sm sm:text-base ${
                  step === 'device-for-case' ? 'bg-accent-red' : 'bg-gray-700'
                }`}>
                  2
                </div>
                <span className="text-sm sm:text-base font-semibold">{t('scan.case.steps.devices')}</span>
              </div>
            </div>
          )}

          {/* Active Case Info Panel */}
          {action === 'case' && step === 'device-for-case' && scannedCase && (
            <div className="mb-4 p-4 rounded-xl bg-accent-red/10 border border-accent-red/30">
              <div className="flex items-center justify-between gap-2 mb-2">
                <div className="flex items-center gap-2 min-w-0">
                  <Box className="w-4 h-4 text-accent-red flex-shrink-0" />
                  <span className="font-semibold text-white text-sm truncate">{scannedCase.name}</span>
                  <span className="text-xs text-gray-400 font-mono flex-shrink-0">#{scannedCase.case_id}</span>
                </div>
                <button
                  type="button"
                  disabled={loading}
                  onClick={() => { setStep('case'); setScannedCase(null); setCaseDeviceIds([]); setCaseActionMessage(null); }}
                  className="flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-semibold bg-white/10 hover:bg-white/20 text-gray-300 transition-colors flex-shrink-0 disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  <X className="w-3 h-3" />
                  {t('scan.case.changeCase')}
                </button>
              </div>
              {caseActionMessage && (
                <div className={`mb-2 flex items-center gap-2 text-xs px-2 py-1.5 rounded-lg ${
                  caseActionMessage.type === 'success' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                }`}>
                  {caseActionMessage.type === 'success'
                    ? <CheckCircle className="w-3.5 h-3.5 flex-shrink-0" />
                    : <XCircle className="w-3.5 h-3.5 flex-shrink-0" />
                  }
                  {caseActionMessage.text}
                </div>
              )}
              {caseDeviceIds.length > 0 && (
                <div className="mt-2">
                  <p className="text-xs text-gray-400 mb-1.5">{t('scan.case.addedDevices', { count: caseDeviceIds.length })}</p>
                  <div className="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto">
                    {caseDeviceIds.map(id => (
                      <span key={id} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-white/10 text-xs text-gray-300 font-mono">
                        {id}
                        <button
                          type="button"
                          disabled={loading}
                          onClick={async () => {
                            try {
                              await casesApi.removeDevice(scannedCase.case_id, id);
                              setCaseDeviceIds(prev => prev.filter(d => d !== id));
                              setCaseActionMessage({ type: 'success', text: t('casesPage.messages.deviceRemoved', { id }) });
                              scheduleCaseActionDismiss(2000);
                            } catch {
                              setCaseActionMessage({ type: 'error', text: t('scan.case.removeFailed') });
                              scheduleCaseActionDismiss(2000);
                            }
                          }}
                          className="text-gray-500 hover:text-red-400 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        >
                          <Trash2 className="w-2.5 h-2.5" />
                        </button>
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Input Method Selector */}
          <div className="flex gap-2 mb-4 sm:mb-6 p-1 bg-white/5 rounded-xl">
            <button
              type="button"
              onClick={() => handleInputMethodChange('keyboard')}
              className={`flex-1 flex items-center justify-center gap-1.5 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm font-semibold transition-all ${
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
                className={`flex-1 flex items-center justify-center gap-1.5 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm font-semibold transition-all ${
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
                className={`flex-1 flex items-center justify-center gap-1.5 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm font-semibold transition-all ${
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
            <div className="mb-4 sm:mb-6">
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
                  {/* Scanning overlay */}
                  <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                    <div className="w-2/3 h-2/3 border-2 border-accent-red/70 rounded-lg relative">
                      <div className="absolute top-0 left-0 w-6 h-6 border-t-4 border-l-4 border-accent-red rounded-tl-md -translate-x-0.5 -translate-y-0.5"></div>
                      <div className="absolute top-0 right-0 w-6 h-6 border-t-4 border-r-4 border-accent-red rounded-tr-md translate-x-0.5 -translate-y-0.5"></div>
                      <div className="absolute bottom-0 left-0 w-6 h-6 border-b-4 border-l-4 border-accent-red rounded-bl-md -translate-x-0.5 translate-y-0.5"></div>
                      <div className="absolute bottom-0 right-0 w-6 h-6 border-b-4 border-r-4 border-accent-red rounded-br-md translate-x-0.5 translate-y-0.5"></div>
                    </div>
                  </div>
                  {!barcodeScanner.isScanning && (
                    <div className="absolute inset-0 flex items-center justify-center bg-black/60">
                      <p className="text-white text-sm">{t('scan.camera.starting')}</p>
                    </div>
                  )}
                </div>
              )}
              <p className="text-center text-gray-400 text-xs sm:text-sm mt-2">
                {t('scan.camera.hint')}
              </p>
            </div>
          )}

          {/* NFC Waiting State */}
          {inputMethod === 'nfc' && (
            <div className="mb-4 sm:mb-6">
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
                  <p className="text-white text-sm sm:text-base font-semibold">
                    {nfcScanner.isScanning ? t('scan.nfc.ready') : t('scan.nfc.starting')}
                  </p>
                  <p className="text-gray-400 text-xs sm:text-sm text-center">
                    {t('scan.nfc.hint')}
                  </p>
                </div>
              )}
            </div>
          )}

          <form onSubmit={handleScan} className="space-y-4 sm:space-y-6">
            {/* Scan Input - primary input in keyboard mode, manual fallback in camera/NFC modes */}
            <div>
              <input
                type="text"
                value={scanCode}
                onChange={(e) => setScanCode(e.target.value)}
                placeholder={
                  inputMethod === 'keyboard'
                    ? step === 'zone'
                      ? t('scan.placeholders.zone')
                      : step === 'case'
                        ? t('scan.case.placeholderCase')
                        : step === 'device-for-case'
                          ? t('scan.case.placeholderDevice')
                          : t('scan.placeholders.device')
                    : t('scan.placeholders.manualFallback')
                }
                autoFocus={inputMethod === 'keyboard'}
                className="w-full px-4 sm:px-6 py-3 sm:py-4 bg-white/10 backdrop-blur-md border-2 border-white/20 rounded-xl text-white text-base sm:text-xl placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
              />
            </div>

            {/* Action Selection - only show in step 1 */}
            {(step === 'device' || step === 'case') && (
              <div className="grid grid-cols-4 gap-2 sm:gap-3">
                {([
                  { value: 'check', label: t('scan.actions.check') },
                  { value: 'intake', label: t('scan.actions.intake') },
                  { value: 'outtake', label: t('scan.actions.outtake') },
                  { value: 'case', label: t('scan.actions.case') },
                ] as const).map((btn) => (
                  <button
                    key={btn.value}
                    type="button"
                    onClick={() => handleActionChange(btn.value)}
                    className={`px-2 sm:px-4 py-2 sm:py-3 rounded-lg sm:rounded-xl text-xs sm:text-sm font-semibold transition-all ${
                      action === btn.value
                        ? 'bg-accent-red text-white scale-105'
                        : 'glass text-gray-400 hover:text-white hover:scale-105'
                    }`}
                  >
                    {btn.label}
                  </button>
                ))}
              </div>
            )}

            {/* Submit Button */}
            <button
              type="submit"
              disabled={loading || !scanCode.trim()}
              className="w-full py-3 sm:py-4 bg-gradient-to-r from-accent-red to-red-700 text-white font-bold text-base sm:text-lg rounded-xl hover:shadow-lg hover:shadow-accent-red/50 disabled:opacity-50 disabled:cursor-not-allowed transition-all transform hover:scale-105 active:scale-95"
            >
              {loading
                ? t('scan.scanning')
                : step === 'zone'
                  ? t('scan.scanZone')
                  : step === 'case'
                    ? t('scan.case.scanCase')
                    : step === 'device-for-case'
                      ? t('scan.case.scanDeviceForCase')
                      : t('scan.scanDevice')}
            </button>
          </form>
        </div>

        {/* Scan Result */}
        {result && (
          <div className={`mt-4 sm:mt-6 glass rounded-xl sm:rounded-2xl p-4 sm:p-6 border-2 ${
            result.success ? 'border-green-500/50' : 'border-red-500/50'
          } animate-fade-in`}>
            <div className="flex items-start gap-3 sm:gap-4">
              {result.success ? (
                <CheckCircle className="w-6 h-6 sm:w-8 sm:h-8 text-green-500 flex-shrink-0" />
              ) : (
                <XCircle className="w-6 h-6 sm:w-8 sm:h-8 text-red-500 flex-shrink-0" />
              )}
              <div className="flex-1 min-w-0">
                <p className={`text-base sm:text-lg font-semibold ${
                  result.success ? 'text-green-400' : 'text-red-400'
                }`}>
                  {result.message}
                </p>
                {result.device && (
                  <div className="mt-2 sm:mt-3 space-y-1.5 sm:space-y-2 text-xs sm:text-sm">
                    <p className="text-gray-300 truncate">
                      <span className="text-gray-500">{t('scan.result.device')}</span> {result.device.product_name}
                    </p>
                    <p className="text-gray-300 truncate">
                      <span className="text-gray-500">{t('scan.result.id')}</span> {result.device.device_id}
                    </p>
                    <p className="text-gray-300">
                      <span className="text-gray-500">{t('scan.result.status')}</span>{' '}
                      <span className={result.success ? 'text-green-400' : 'text-yellow-400'}>
                        {result.new_status || result.device.status}
                      </span>
                    </p>
                  </div>
                )}
                {/* Quick service actions after a successful check scan */}
                {result.success && result.action === 'check' && result.device && (
                  <div className="mt-3 sm:mt-4 flex flex-col sm:flex-row gap-2">
                    <button
                      type="button"
                      onClick={() => setViewDevice(result.device!)}
                      className="flex items-center justify-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm font-semibold bg-blue-500/20 text-blue-300 border border-blue-500/30 hover:bg-blue-500/30 transition-all"
                    >
                      <Info className="w-4 h-4 flex-shrink-0" />
                      {t('scan.serviceActions.moreInfo')}
                    </button>
                    <button
                      type="button"
                      onClick={() => openServiceModal(result.device!.device_id, 'medium')}
                      className="flex items-center justify-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm font-semibold bg-yellow-500/20 text-yellow-300 border border-yellow-500/30 hover:bg-yellow-500/30 transition-all"
                    >
                      <Wrench className="w-4 h-4 flex-shrink-0" />
                      {t('scan.serviceActions.reportIssue')}
                    </button>
                    <button
                      type="button"
                      onClick={() => openServiceModal(result.device!.device_id, 'high')}
                      className="flex items-center justify-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm font-semibold bg-red-500/20 text-red-300 border border-red-500/30 hover:bg-red-500/30 transition-all"
                    >
                      <AlertTriangle className="w-4 h-4 flex-shrink-0" />
                      {t('scan.serviceActions.markProblematic')}
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Service / Defect Report Modal */}
        {showServiceModal && (
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark rounded-2xl p-6 sm:p-8 border-2 border-white/10 max-w-md w-full">
              {serviceSuccess ? (
                <div className="text-center py-4">
                  <div className="inline-block p-4 rounded-xl bg-green-500/20 mb-4">
                    <CheckCircle className="w-12 h-12 text-green-400" />
                  </div>
                  <h2 className="text-2xl font-bold text-white mb-2">{t('scan.serviceModal.successTitle')}</h2>
                  <p className="text-gray-400 text-sm sm:text-base mb-6">
                    {t('scan.serviceModal.successDescription')}
                  </p>
                  <button
                    onClick={handleServiceModalClose}
                    className="w-full px-4 py-3 rounded-lg font-semibold bg-gradient-to-r from-accent-red to-red-700 text-white hover:shadow-lg hover:shadow-accent-red/50 transition-all"
                  >
                    {t('common.close')}
                  </button>
                </div>
              ) : (
                <>
                  <div className="text-center mb-6">
                    <div className={`inline-block p-4 rounded-xl mb-4 ${
                      serviceForm.severity === 'high' || serviceForm.severity === 'critical'
                        ? 'bg-red-500/20'
                        : 'bg-yellow-500/20'
                    }`}>
                      {serviceForm.severity === 'high' || serviceForm.severity === 'critical' ? (
                        <AlertTriangle className="w-10 h-10 text-red-400" />
                      ) : (
                        <Wrench className="w-10 h-10 text-yellow-300" />
                      )}
                    </div>
                    <h2 className="text-xl sm:text-2xl font-bold text-white mb-1">
                      {t('scan.serviceModal.title')}
                    </h2>
                    <p className="text-gray-400 text-sm">{t('scan.serviceModal.subtitle')}</p>
                  </div>

                  <form onSubmit={handleServiceSubmit} className="space-y-3 sm:space-y-4">
                    {/* Device ID (read-only) */}
                    <div>
                      <label className="block text-xs sm:text-sm font-medium text-gray-400 mb-1.5">
                        {t('maintenance.form.deviceId')}
                      </label>
                      <input
                        type="text"
                        value={serviceForm.device_id}
                        readOnly
                        className="w-full px-3 sm:px-4 py-2 sm:py-2.5 bg-white/5 border-2 border-white/10 rounded-lg sm:rounded-xl text-sm sm:text-base text-gray-300 cursor-default"
                      />
                    </div>

                    {/* Severity */}
                    <div>
                      <label className="block text-xs sm:text-sm font-medium text-gray-400 mb-1.5">
                        {t('maintenance.form.severity')}
                      </label>
                      <select
                        value={serviceForm.severity}
                        onChange={(e) => setServiceForm({ ...serviceForm, severity: e.target.value })}
                        className="w-full px-3 sm:px-4 py-2 sm:py-2.5 bg-white/10 border-2 border-white/20 rounded-lg sm:rounded-xl text-sm sm:text-base text-white focus:outline-none focus:border-accent-red"
                      >
                        <option value="low">{t('maintenance.severity.low')}</option>
                        <option value="medium">{t('maintenance.severity.medium')}</option>
                        <option value="high">{t('maintenance.severity.high')}</option>
                        <option value="critical">{t('maintenance.severity.critical')}</option>
                      </select>
                    </div>

                    {/* Title */}
                    <div>
                      <label className="block text-xs sm:text-sm font-medium text-gray-400 mb-1.5">
                        {t('maintenance.form.title')}
                      </label>
                      <input
                        type="text"
                        value={serviceForm.title}
                        onChange={(e) => setServiceForm({ ...serviceForm, title: e.target.value })}
                        required
                        placeholder={t('scan.serviceModal.titlePlaceholder')}
                        className="w-full px-3 sm:px-4 py-2 sm:py-2.5 bg-white/10 border-2 border-white/20 rounded-lg sm:rounded-xl text-sm sm:text-base text-white placeholder-gray-500 focus:outline-none focus:border-accent-red"
                      />
                    </div>

                    {/* Description */}
                    <div>
                      <label className="block text-xs sm:text-sm font-medium text-gray-400 mb-1.5">
                        {t('maintenance.form.description')}
                      </label>
                      <textarea
                        value={serviceForm.description}
                        onChange={(e) => setServiceForm({ ...serviceForm, description: e.target.value })}
                        required
                        rows={3}
                        placeholder={t('scan.serviceModal.descriptionPlaceholder')}
                        className="w-full px-3 sm:px-4 py-2 sm:py-2.5 bg-white/10 border-2 border-white/20 rounded-lg sm:rounded-xl text-sm sm:text-base text-white placeholder-gray-500 focus:outline-none focus:border-accent-red resize-none"
                      />
                    </div>

                    <div className="flex gap-3 pt-1">
                      <button
                        type="button"
                        onClick={handleServiceModalClose}
                        disabled={serviceLoading}
                        className="flex-1 px-4 py-2.5 sm:py-3 rounded-lg font-semibold bg-white/10 text-white hover:bg-white/20 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-sm sm:text-base"
                      >
                        {t('common.cancel')}
                      </button>
                      <button
                        type="submit"
                        disabled={serviceLoading}
                        className="flex-1 px-4 py-2.5 sm:py-3 rounded-lg font-semibold bg-gradient-to-r from-accent-red to-red-700 text-white hover:shadow-lg hover:shadow-accent-red/50 disabled:opacity-50 disabled:cursor-not-allowed transition-all text-sm sm:text-base"
                      >
                        {serviceLoading ? t('common.saving') : t('maintenance.form.submit')}
                      </button>
                    </div>
                    {serviceError && (
                      <p className="text-red-400 text-xs sm:text-sm text-center pt-1">{serviceError}</p>
                    )}
                  </form>
                </>
              )}
            </div>
          </div>
        )}

        {showLEDModal && (
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="flex justify-center">
              <div className="glass-dark rounded-2xl p-6 sm:p-8 border-2 border-white/10 max-w-md w-full">
              <div className="text-center mb-6">
                <div className="inline-block p-4 rounded-xl bg-yellow-500/20 mb-4">
                  <Lightbulb className="w-12 h-12 text-yellow-300" />
                </div>
                <h2 className="text-2xl font-bold text-white mb-2">{t('scan.ledModal.title')}</h2>
                <p className="text-gray-400 text-sm sm:text-base">
                  {t('scan.ledModal.description')}
                </p>
              </div>

              <div className="flex gap-3">
                <button
                  onClick={handleLEDModalCancel}
                  className="flex-1 px-4 py-3 rounded-lg font-semibold bg-white/10 text-white hover:bg-white/20 transition-colors"
                >
                  {t('scan.ledModal.cancel')}
                </button>
                <button
                  onClick={handleLEDModalConfirm}
                  disabled={loading}
                  className="flex-1 px-4 py-3 rounded-lg font-semibold bg-gradient-to-r from-accent-red to-red-700 text-white hover:shadow-lg hover:shadow-accent-red/50 disabled:opacity-50 transition-all"
                >
                  {t('scan.ledModal.confirm')}
                </button>
              </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Device Info Modal */}
      <DeviceInfoModal
        device={viewDevice}
        isOpen={viewDevice !== null}
        onClose={() => setViewDevice(null)}
        onEdit={(device) => {
          setViewDevice(null);
          navigate('/products', { state: { openEditDeviceId: device.device_id } });
        }}
      />
    </div>
  );
}
