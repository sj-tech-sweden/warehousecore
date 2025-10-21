import { useState, useEffect } from 'react';
import { Package, CheckCircle, XCircle, Calendar, User, ArrowRight, Lightbulb, LightbulbOff } from 'lucide-react';
import { jobsApi, scansApi, ledApi } from '../lib/api';
import type { Job, JobSummary, JobDevice, LEDStatus } from '../lib/api';

const JOB_CODE_PATTERN = /^JOB\d+$/i;

export function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [selectedJob, setSelectedJob] = useState<JobSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [scanCode, setScanCode] = useState('');
  const [scanLoading, setScanLoading] = useState(false);
  const [scanResult, setScanResult] = useState<{ success: boolean; message: string } | null>(null);

  // LED state
  const [ledActive, setLedActive] = useState(false);
  const [ledStatus, setLedStatus] = useState<LEDStatus | null>(null);
  const [ledLoading, setLedLoading] = useState(false);

  // Load open jobs and LED status on mount
  useEffect(() => {
    loadJobs();
    loadLEDStatus();
  }, []);

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

  const handleScan = async (e: React.FormEvent) => {
    e.preventDefault();
    const rawCode = scanCode.trim();
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
          throw new Error('Ungültige Job-ID');
        }

        await loadJobDetails(numericPart, { highlight: true });
        setScanResult({ success: true, message: `Job ${normalizedCode} geladen` });
      } catch (error: any) {
        console.error('Job scan failed:', error);
        setScanResult({
          success: false,
          message: error.response?.data?.error || error.message || 'Job nicht gefunden',
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
        message: 'Bitte zuerst einen Job auswählen oder scannen.',
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
        scan_code: scanCode,
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
        message: error.response?.data?.error || 'Scan fehlgeschlagen',
      });
    } finally {
      setScanLoading(false);
      // Clear result after 3 seconds
      setTimeout(() => setScanResult(null), 3000);
    }
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
      alert(error.response?.data?.error || 'LED-Steuerung fehlgeschlagen');
    } finally {
      setLedLoading(false);
    }
  };

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
            <h1 className="text-4xl font-bold text-white mb-2">Offene Jobs</h1>
            <p className="text-gray-400">Wähle einen Job zum Ausscannen aus dem Lager</p>
          </div>

          {loading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red"></div>
              <p className="text-gray-400 mt-4">Lade Jobs...</p>
            </div>
          ) : jobs.length === 0 ? (
            <div className="glass-dark rounded-2xl p-12 text-center">
              <Package className="w-16 h-16 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-400 text-lg">Keine offenen Jobs vorhanden</p>
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
                      {job.status}
                    </span>
                  </div>

                  <h3 className="text-xl font-bold text-white mb-2">
                    Job {job.job_code}
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
                      <span>{job.device_count} Geräte</span>
                    </div>
                  </div>

                  <div className="mt-4 flex items-center gap-2 text-accent-red group-hover:gap-3 transition-all">
                    <span className="font-semibold">Auswählen</span>
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
            Zurück zur Job-Liste
          </button>

          <div className="glass-dark rounded-2xl p-6 border-2 border-white/10">
            <div className="flex items-start justify-between mb-4">
              <div>
                <h1 className="text-3xl font-bold text-white mb-2">Job {selectedJob.job_code}</h1>
                {selectedJob.description && (
                  <p className="text-gray-400">{selectedJob.description}</p>
                )}
              </div>
              <span className="px-4 py-2 rounded-full bg-green-500/20 text-green-400 font-semibold">
                {selectedJob.status}
              </span>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
              {(selectedJob.customer_first_name || selectedJob.customer_last_name) && (
                <div className="flex items-center gap-3">
                  <User className="w-5 h-5 text-gray-500" />
                  <div>
                    <p className="text-xs text-gray-500">Kunde</p>
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
                    <p className="text-xs text-gray-500">Datum</p>
                    <p className="text-white font-semibold">
                      {new Date(selectedJob.start_date).toLocaleDateString('de-DE')}
                    </p>
                  </div>
                </div>
              )}

              <div className="flex items-center gap-3">
                <Package className="w-5 h-5 text-gray-500" />
                <div>
                  <p className="text-xs text-gray-500">Fortschritt</p>
                  <p className="text-white font-semibold">
                    {stats.scanned} / {stats.total} Geräte
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
              <span className="text-gray-400">{progress.toFixed(0)}% ausgescannt</span>
              <span className="text-gray-400">{stats.remaining} verbleibend</span>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Scan Interface */}
          <div className="glass-dark rounded-2xl p-6 border-2 border-white/10">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-2xl font-bold text-white">Gerät Ausscannen</h2>
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
                    <span>Fächer hervorgehoben</span>
                    <LightbulbOff className="w-5 h-5 ml-auto" />
                  </>
                ) : (
                  <>
                    <LightbulbOff className="w-5 h-5" />
                    <span>Fächer hervorheben</span>
                    <Lightbulb className="w-5 h-5 ml-auto" />
                  </>
                )}
              </button>

              {/* LED Status Info */}
              {ledStatus && (
                <div className="mt-2 flex items-center justify-between text-xs">
                  <span className={`flex items-center gap-1 ${ledStatus.mqtt_connected ? 'text-green-400' : 'text-gray-500'}`}>
                    <span className={`w-2 h-2 rounded-full ${ledStatus.mqtt_connected ? 'bg-green-400' : 'bg-gray-500'}`}></span>
                    {ledStatus.mqtt_connected ? 'MQTT verbunden' : ledStatus.mqtt_dry_run ? 'Dry-Run Modus' : 'MQTT nicht konfiguriert'}
                  </span>
                  {ledStatus.mapping_loaded && (
                    <span className="text-gray-400">
                      {ledStatus.total_bins} Fächer verfügbar
                    </span>
                  )}
                </div>
              )}
            </div>

            <form onSubmit={handleScan} className="space-y-4">
              <input
                type="text"
                value={scanCode}
                onChange={(e) => setScanCode(e.target.value)}
                placeholder="Barcode / QR-Code scannen"
                autoFocus
                className="w-full px-6 py-4 bg-white/10 backdrop-blur-md border-2 border-white/20 rounded-xl text-white text-xl placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
              />

              <button
                type="submit"
                disabled={scanLoading || !scanCode.trim()}
                className="w-full py-4 bg-gradient-to-r from-accent-red to-red-700 text-white font-bold text-lg rounded-xl hover:shadow-lg hover:shadow-accent-red/50 disabled:opacity-50 disabled:cursor-not-allowed transition-all transform hover:scale-105 active:scale-95"
              >
                {scanLoading ? 'Scanne...' : 'Gerät Ausscannen'}
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
            <h2 className="text-2xl font-bold text-white mb-4">Geräte-Liste</h2>

            {selectedJob.devices.length === 0 ? (
              <p className="text-gray-400 text-center py-8">Keine Geräte in diesem Job</p>
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
                        <p className="text-sm text-gray-400">ID: {device.device_id}</p>
                        {device.zone_name && (
                          <p className="text-sm text-gray-500">Lager: {device.zone_name}</p>
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
                            {device.status}
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
