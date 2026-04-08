import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams } from 'react-router-dom';
import { Package, CheckCircle, XCircle, Calendar, User, ArrowRight, Lightbulb, LightbulbOff, Camera, Nfc, Keyboard } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { jobsApi, scansApi, ledApi } from '../lib/api';
import type { Job, JobSummary, JobDevice, LEDStatus } from '../lib/api';
import { useBarcodeScanner } from '../hooks/useBarcodeScanner';
import { useNFCScanner } from '../hooks/useNFCScanner';

const JOB_CODE_PATTERN = /^JOB\d+$/i;

type InputMethod = 'keyboard' | 'camera' | 'nfc';

export function JobsPage() {
  const { t } = useTranslation();
  const { id: urlJobId } = useParams<{ id: string }>();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [selectedJob, setSelectedJob] = useState<JobSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [scanCode, setScanCode] = useState('');
  const [scanLoading, setScanLoading] = useState(false);
  const [scanResult, setScanResult] = useState<{ success: boolean; message: string } | null>(null);

  // Input method for the scan card: keyboard (default), camera, or nfc
  const [inputMethod, setInputMethod] = useState<InputMethod>('keyboard');

  // Stable ref so camera/NFC callbacks can always reach the latest scan handler
  const processCodeRef = useRef<(code: string) => void>(() => {});

  // LED state
  const [ledActive, setLedActive] = useState(false);
  const [ledStatus, setLedStatus] = useState<LEDStatus | null>(null);
  const [ledLoading, setLedLoading] = useState(false);

  // Load open jobs and LED status on mount
  useEffect(() => {
    loadJobs();
    loadLEDStatus();
  }, []);

  // Load job from URL parameter if present
  useEffect(() => {
    if (urlJobId) {
      const jobId = parseInt(urlJobId, 10);
      if (!isNaN(jobId)) {
        loadJobDetails(jobId, { highlight: true });
      }
    }
  }, [urlJobId]);

  const loadLEDStatus = async () => {
    try {
      const { data } = await ledApi.getStatus();
      setLedStatus(data);
    } catch (error) {
      console.error('Failed to load LED status:', error);
    }
  };

  // Auto-refresh job details when selected
  useEffect(() => {
    if (selectedJob) {
      const interval = setInterval(() => {
        refreshJobDetails(selectedJob.job_id);
      }, 2000); // Refresh every 2 seconds for live updates

      return () => clearInterval(interval);
    }
  }, [selectedJob]);

  // Cleanup LEDs when leaving the page or unmounting
  useEffect(() => {
    const clearLEDsOnExit = async () => {
      if (ledActive) {
        try {
          await ledApi.clear();
        } catch (error) {
          console.error('Failed to clear LEDs on exit:', error);
        }
      }
    };

    // Cleanup when component unmounts (navigating to different page)
    return () => {
      clearLEDsOnExit();
    };
  }, [ledActive]);

  // Clear LEDs when browser is closed or page is reloaded
  useEffect(() => {
    const handleBeforeUnload = async () => {
      if (ledActive) {
        try {
          // Use navigator.sendBeacon for reliable cleanup on page unload
          await ledApi.clear();
        } catch (error) {
          console.error('Failed to clear LEDs on unload:', error);
        }
      }
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => window.removeEventListener('beforeunload', handleBeforeUnload);
  }, [ledActive]);

  const loadJobs = async () => {
    try {
      setLoading(true);
      const { data } = await jobsApi.getAll({ status: 'open' });
      setJobs(data);
    } catch (error) {
      console.error('Failed to load jobs:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadJobDetails = async (jobId: number, options: { highlight?: boolean } = {}) => {
    try {
      setLoading(true);
      const { data } = await jobsApi.getById(jobId);
      setSelectedJob(data);

      if (options.highlight !== false) {
        setLedActive(false);
        try {
          await ledApi.highlightJob(jobId);
          setLedActive(true);
        } catch (error) {
          console.error('Failed to highlight job LEDs:', error);
          setLedActive(false);
        }
      }
    } catch (error) {
      console.error('Failed to load job details:', error);
    } finally {
      setLoading(false);
    }
  };

  const refreshJobDetails = async (jobId: number) => {
    try {
      const { data } = await jobsApi.getById(jobId);
      setSelectedJob(data);
    } catch (error) {
      console.error('Failed to refresh job:', error);
    }
  };

  const handleCodeDetected = useCallback((code: string) => {
    processCodeRef.current(code);
  }, []);

  const barcodeScanner = useBarcodeScanner({ onDetected: handleCodeDetected });
  const nfcScanner = useNFCScanner({ onDetected: handleCodeDetected });

  const handleInputMethodChange = useCallback((method: InputMethod) => {
    setScanCode('');
    setInputMethod(method);
  }, []);

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

  // Reset input method and stop scanners when leaving job detail view
  useEffect(() => {
    if (!selectedJob) {
      barcodeScanner.stopScanning();
      nfcScanner.stopScanning();
      setInputMethod('keyboard');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedJob]);

  // Stop scanners on unmount
  useEffect(() => {
    return () => {
      barcodeScanner.stopScanning();
      nfcScanner.stopScanning();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Keep ref pointing to the latest processCode so camera/NFC callbacks are never stale
  const processCode = async (code: string) => {
    const rawCode = code.trim();
    if (!rawCode) {
      return;
    }

    const normalizedCode = rawCode.toUpperCase();

    // Detect job code scans (e.g., JOB0001)
    if (JOB_CODE_PATTERN.test(normalizedCode)) {
      setScanLoading(true);
      setScanResult(null);

      try {
        const numericPart = parseInt(normalizedCode.replace(/\D/g, ''), 10);
        if (Number.isNaN(numericPart)) {
          throw new Error(t('jobsPage.invalidJobId'));
        }

        await loadJobDetails(numericPart, { highlight: true });
        setScanResult({ success: true, message: t('jobsPage.jobLoaded', { code: normalizedCode }) });
      } catch (error: any) {
        console.error('Job scan failed:', error);
        setScanResult({
          success: false,
          message: error.response?.data?.error || error.message || t('jobsPage.jobNotFoundGeneric'),
        });
      } finally {
        setScanCode('');
        setScanLoading(false);
        setTimeout(() => setScanResult(null), 3000);
      }

      return;
    }

    if (!selectedJob) {
      setScanResult({
        success: false,
        message: t('jobsPage.selectJobFirst'),
      });
      setScanLoading(false);
      setTimeout(() => setScanResult(null), 3000);
      return;
    }

    setScanLoading(true);
    setScanResult(null);

    try {
      // Process outtake scan with job context
      const { data } = await scansApi.process({
        scan_code: code,
        action: 'outtake',
        job_id: selectedJob.job_id,
      });

      setScanResult({
        success: data.success,
        message: data.message,
      });

      setScanCode('');

      // Refresh job details immediately after scan
      if (data.success) {
        await refreshJobDetails(selectedJob.job_id);
      }
    } catch (error: any) {
      console.error('Scan failed:', error);
      setScanResult({
        success: false,
        message: error.response?.data?.error || t('scan.scanError'),
      });
    } finally {
      setScanLoading(false);
      // Clear result after 3 seconds
      setTimeout(() => setScanResult(null), 3000);
    }
  };

  // Update ref on every render so camera/NFC callbacks always use the latest closure
  processCodeRef.current = processCode;

  const handleScan = (e: React.FormEvent) => {
    e.preventDefault();
    processCode(scanCode);
  };

  const handleBackToList = async () => {
    // Turn off LEDs when leaving job
    if (ledActive) {
      try {
        await ledApi.clear();
        setLedActive(false);
      } catch (error) {
        console.error('Failed to clear LEDs:', error);
      }
    }

    setSelectedJob(null);
    setScanCode('');
    setScanResult(null);
    loadJobs(); // Reload job list
  };

  const toggleLEDHighlight = async () => {
    if (!selectedJob) return;

    setLedLoading(true);
    try {
      if (ledActive) {
        // Turn off LEDs
        await ledApi.clear();
        setLedActive(false);
      } else {
        // Turn on LEDs for this job
        await ledApi.highlightJob(selectedJob.job_id);
        setLedActive(true);
      }
    } catch (error: any) {
      console.error('LED toggle failed:', error);
      alert(error.response?.data?.error || t('jobsPage.ledToggleError'));
    } finally {
      setLedLoading(false);
    }
  };

  const formatJobStatus = (status: string) => t(`jobs.statuses.${status}`, status);
  const formatDeviceStatus = (status: string) => t(`devices.statuses.${status}`, status);

  const getDeviceStats = (devices: JobDevice[]) => {
    const total = devices.length;
    const scanned = devices.filter(d => d.scanned).length;
    const remaining = total - scanned;
    return { total, scanned, remaining };
  };

  // Job List View
  if (!selectedJob) {
    return (
      <div className="min-h-screen p-6">
        <div className="max-w-6xl mx-auto">
          <div className="mb-8">
            <h1 className="text-4xl font-bold text-white mb-2">{t('jobsPage.openJobsTitle')}</h1>
            <p className="text-gray-400">{t('jobsPage.openJobsSubtitle')}</p>
          </div>

          {loading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red"></div>
              <p className="text-gray-400 mt-4">{t('jobsPage.loadingJobs')}</p>
            </div>
          ) : jobs.length === 0 ? (
            <div className="glass-dark rounded-2xl p-12 text-center">
              <Package className="w-16 h-16 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-400 text-lg">{t('jobsPage.noOpenJobs')}</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {jobs.map((job) => (
                <button
                  key={job.job_id}
                  onClick={() => loadJobDetails(job.job_id, { highlight: false })}
                  className="glass-dark rounded-2xl p-6 border-2 border-white/10 hover:border-accent-red transition-all text-left group hover:scale-105"
                >
                  <div className="flex items-start justify-between mb-4">
                    <div className="p-3 rounded-xl bg-gradient-to-br from-accent-red to-red-700">
                      <Package className="w-6 h-6 text-white" />
                    </div>
                    <span className="px-3 py-1 rounded-full bg-green-500/20 text-green-400 text-sm font-semibold">
                      {formatJobStatus(job.status)}
                    </span>
                  </div>

                  <h3 className="text-xl font-bold text-white mb-2">
                    {t('jobsPage.jobTitle', { code: job.job_code })}
                  </h3>

                  {job.description && (
                    <p className="text-gray-400 mb-3 line-clamp-2">{job.description}</p>
                  )}

                  <div className="space-y-2 text-sm">
                    {(job.customer_first_name || job.customer_last_name) && (
                      <div className="flex items-center gap-2 text-gray-400">
                        <User className="w-4 h-4" />
                        <span>{job.customer_first_name} {job.customer_last_name}</span>
                      </div>
                    )}

                    {job.start_date && (
                      <div className="flex items-center gap-2 text-gray-400">
                        <Calendar className="w-4 h-4" />
                        <span>{new Date(job.start_date).toLocaleDateString('de-DE')}</span>
                      </div>
                    )}

                    <div className="flex items-center gap-2 text-accent-red font-semibold">
                      <Package className="w-4 h-4" />
                      <span>{t('jobsPage.deviceCount', { count: job.device_count })}</span>
                    </div>
                  </div>

                  <div className="mt-4 flex items-center gap-2 text-accent-red group-hover:gap-3 transition-all">
                    <span className="font-semibold">{t('jobsPage.select')}</span>
                    <ArrowRight className="w-4 h-4" />
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    );
  }

  // Job Details & Scan View
  const stats = getDeviceStats(selectedJob.devices);
  const progress = stats.total > 0 ? (stats.scanned / stats.total) * 100 : 0;

  return (
    <div className="min-h-screen p-6">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="mb-6">
          <button
            onClick={handleBackToList}
            className="text-gray-400 hover:text-white mb-4 flex items-center gap-2"
          >
            <ArrowRight className="w-4 h-4 rotate-180" />
            {t('jobsPage.backToList')}
          </button>

          <div className="glass-dark rounded-2xl p-6 border-2 border-white/10">
            <div className="flex items-start justify-between mb-4">
              <div>
                <h1 className="text-3xl font-bold text-white mb-2">{t('jobsPage.jobTitle', { code: selectedJob.job_code })}</h1>
                {selectedJob.description && (
                  <p className="text-gray-400">{selectedJob.description}</p>
                )}
              </div>
              <span className="px-4 py-2 rounded-full bg-green-500/20 text-green-400 font-semibold">
                {formatJobStatus(selectedJob.status)}
              </span>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
              {(selectedJob.customer_first_name || selectedJob.customer_last_name) && (
                <div className="flex items-center gap-3">
                  <User className="w-5 h-5 text-gray-500" />
                  <div>
                    <p className="text-xs text-gray-500">{t('jobs.customer')}</p>
                    <p className="text-white font-semibold">
                      {selectedJob.customer_first_name} {selectedJob.customer_last_name}
                    </p>
                  </div>
                </div>
              )}

              {selectedJob.start_date && (
                <div className="flex items-center gap-3">
                  <Calendar className="w-5 h-5 text-gray-500" />
                  <div>
                    <p className="text-xs text-gray-500">{t('jobsPage.date')}</p>
                    <p className="text-white font-semibold">
                      {new Date(selectedJob.start_date).toLocaleDateString('de-DE')}
                    </p>
                  </div>
                </div>
              )}

              <div className="flex items-center gap-3">
                <Package className="w-5 h-5 text-gray-500" />
                <div>
                  <p className="text-xs text-gray-500">{t('jobsPage.progress')}</p>
                  <p className="text-white font-semibold">
                    {t('jobsPage.progressValue', { scanned: stats.scanned, total: stats.total })}
                  </p>
                </div>
              </div>
            </div>

            {/* Progress Bar */}
            <div className="w-full bg-gray-700 rounded-full h-3 overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-accent-red to-green-500 transition-all duration-500"
                style={{ width: `${progress}%` }}
              />
            </div>
            <div className="flex justify-between mt-2 text-sm">
              <span className="text-gray-400">{t('jobsPage.scannedPercent', { percent: progress.toFixed(0) })}</span>
              <span className="text-gray-400">{t('jobsPage.remaining', { count: stats.remaining })}</span>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Scan Interface */}
          <div className="glass-dark rounded-2xl p-6 border-2 border-white/10">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-2xl font-bold text-white">{t('jobsPage.outtakeDevice')}</h2>
            </div>

            {/* LED Highlight Toggle */}
            <div className="mb-4">
              <button
                onClick={toggleLEDHighlight}
                disabled={ledLoading}
                className={`w-full py-3 px-4 rounded-xl font-semibold text-white transition-all flex items-center justify-center gap-2 ${
                  ledActive
                    ? 'bg-gradient-to-r from-green-600 to-green-700 hover:shadow-lg hover:shadow-green-500/50'
                    : 'bg-gradient-to-r from-gray-600 to-gray-700 hover:shadow-lg hover:shadow-gray-500/50'
                } ${ledLoading ? 'opacity-50 cursor-not-allowed' : 'hover:scale-105 active:scale-95'}`}
              >
                {ledActive ? (
                  <>
                    <Lightbulb className="w-5 h-5" />
                    <span>{t('jobsPage.ledHighlighted')}</span>
                    <LightbulbOff className="w-5 h-5 ml-auto" />
                  </>
                ) : (
                  <>
                    <LightbulbOff className="w-5 h-5" />
                    <span>{t('jobsPage.highlightBins')}</span>
                    <Lightbulb className="w-5 h-5 ml-auto" />
                  </>
                )}
              </button>

              {/* LED Status Info */}
              {ledStatus && (
                <div className="mt-2 flex items-center justify-between text-xs">
                  <span className={`flex items-center gap-1 ${ledStatus.mqtt_connected ? 'text-green-400' : 'text-gray-500'}`}>
                    <span className={`w-2 h-2 rounded-full ${ledStatus.mqtt_connected ? 'bg-green-400' : 'bg-gray-500'}`}></span>
                    {ledStatus.mqtt_connected ? t('jobsPage.mqttConnected') : ledStatus.mqtt_dry_run ? t('jobsPage.dryRunMode') : t('jobsPage.mqttNotConfigured')}
                  </span>
                  {ledStatus.mapping_loaded && (
                    <span className="text-gray-400">
                      {t('jobsPage.binsAvailable', { count: ledStatus.total_bins })}
                    </span>
                  )}
                </div>
              )}
            </div>

            <div className="flex gap-1 p-1 bg-white/5 rounded-xl mb-4">
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
              <div className="mb-4">
                {barcodeScanner.error ? (
                  <div className="flex items-center gap-3 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
                    <XCircle className="w-5 h-5 flex-shrink-0" />
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
              <div className="mb-4">
                {nfcScanner.error ? (
                  <div className="flex items-center gap-3 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
                    <XCircle className="w-5 h-5 flex-shrink-0" />
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

            <form onSubmit={handleScan} className="space-y-4">
              <input
                type="text"
                value={scanCode}
                onChange={(e) => setScanCode(e.target.value)}
                placeholder={t('jobsPage.scanPlaceholder')}
                autoFocus={inputMethod === 'keyboard'}
                className="w-full px-6 py-4 bg-white/10 backdrop-blur-md border-2 border-white/20 rounded-xl text-white text-xl placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
              />

              <button
                type="submit"
                disabled={scanLoading || !scanCode.trim()}
                className="w-full py-4 bg-gradient-to-r from-accent-red to-red-700 text-white font-bold text-lg rounded-xl hover:shadow-lg hover:shadow-accent-red/50 disabled:opacity-50 disabled:cursor-not-allowed transition-all transform hover:scale-105 active:scale-95"
              >
                {scanLoading ? t('jobsPage.scanning') : t('jobsPage.outtakeDevice')}
              </button>
            </form>

            {/* Scan Result */}
            {scanResult && (
              <div
                className={`mt-4 p-4 rounded-xl border-2 flex items-center gap-3 ${
                  scanResult.success
                    ? 'bg-green-500/10 border-green-500/50'
                    : 'bg-red-500/10 border-red-500/50'
                }`}
              >
                {scanResult.success ? (
                  <CheckCircle className="w-6 h-6 text-green-500 flex-shrink-0" />
                ) : (
                  <XCircle className="w-6 h-6 text-red-500 flex-shrink-0" />
                )}
                <p
                  className={`font-semibold ${
                    scanResult.success ? 'text-green-400' : 'text-red-400'
                  }`}
                >
                  {scanResult.message}
                </p>
              </div>
            )}
          </div>

          {/* Device List */}
          <div className="glass-dark rounded-2xl p-6 border-2 border-white/10 max-h-[600px] overflow-y-auto">
            <h2 className="text-2xl font-bold text-white mb-4">{t('jobsPage.deviceList')}</h2>

            {selectedJob.devices.length === 0 ? (
              <p className="text-gray-400 text-center py-8">{t('jobsPage.noDevicesInJob')}</p>
            ) : (
              <div className="space-y-2">
                {selectedJob.devices.map((device) => (
                  <div
                    key={device.device_id}
                    className={`p-4 rounded-xl border-2 transition-all ${
                      device.scanned
                        ? 'bg-green-500/10 border-green-500/50'
                        : 'bg-white/5 border-white/10'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex-1">
                        <p className="font-semibold text-white">{device.product_name}</p>
                        <p className="text-sm text-gray-400">{t('jobsPage.deviceId', { id: device.device_id })}</p>
                        {device.zone_name && (
                          <p className="text-sm text-gray-500">{t('jobsPage.zone', { zone: device.zone_name })}</p>
                        )}
                      </div>

                      <div className="flex items-center gap-3">
                        <div className="text-right">
                          <span
                            className={`text-xs px-2 py-1 rounded-full font-semibold ${
                              device.status === 'on_job'
                                ? 'bg-blue-500/20 text-blue-400'
                                : device.status === 'in_storage'
                                ? 'bg-green-500/20 text-green-400'
                                : 'bg-gray-500/20 text-gray-400'
                            }`}
                          >
                            {formatDeviceStatus(device.status)}
                          </span>
                        </div>

                        {device.scanned ? (
                          <CheckCircle className="w-8 h-8 text-green-500 flex-shrink-0" />
                        ) : (
                          <XCircle className="w-8 h-8 text-gray-600 flex-shrink-0" />
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
