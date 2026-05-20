import { useState, useEffect, useCallback, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Package, CheckCircle, XCircle, Calendar, User, ArrowRight, Lightbulb, LightbulbOff, ClipboardList, Camera, Nfc, Keyboard, PlusCircle, Pencil, Trash2, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { jobsApi, scansApi, ledApi, devicesApi } from '../lib/api';
import type { Job, JobSummary, JobDevice, LEDStatus, ProductRequirement, JobCreateInput, JobCustomerOption, JobStatusOption, DeviceTreeCategory } from '../lib/api';
import { formatDateISO } from '../lib/utils';
import { useBarcodeScanner } from '../hooks/useBarcodeScanner';
import { useNFCScanner } from '../hooks/useNFCScanner';
import type { InputMethod } from '../types/scanTypes';
import { JobRequirementTree, type JobRequirementSelection } from '../components/JobRequirementTree';

const JOB_CODE_PATTERN = /^JOB\d+$/i;

export function JobsPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id: urlJobId } = useParams<{ id: string }>();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [selectedJob, setSelectedJob] = useState<JobSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [scanCode, setScanCode] = useState('');
  const [scanLoading, setScanLoading] = useState(false);
  const [scanResult, setScanResult] = useState<{ success: boolean; message: string } | null>(null);
  const [scanAction, setScanAction] = useState<'outtake' | 'intake'>('outtake');
  const [requirementTreeData, setRequirementTreeData] = useState<DeviceTreeCategory[]>([]);
  const [requirementTreeLoading, setRequirementTreeLoading] = useState(false);
  const [requirementTreeSearch, setRequirementTreeSearch] = useState('');
  const [requirementTreeExpanded, setRequirementTreeExpanded] = useState<Set<string>>(new Set());
  const [requirementDrafts, setRequirementDrafts] = useState<JobRequirementSelection[]>([]);
  const [savingRequirement, setSavingRequirement] = useState(false);
  const [requirementMessage, setRequirementMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Job create/edit modal state
  const [jobModalOpen, setJobModalOpen] = useState(false);
  const [editingJob, setEditingJob] = useState<Job | null>(null);
  const [jobFormData, setJobFormData] = useState<JobCreateInput>({ job_code: '', description: '', start_date: '', end_date: '' });
  const [jobCustomers, setJobCustomers] = useState<JobCustomerOption[]>([]);
  const [jobStatuses, setJobStatuses] = useState<JobStatusOption[]>([]);
  const [defaultJobStatusID, setDefaultJobStatusID] = useState<number | undefined>(undefined);
  const [jobModalLoading, setJobModalLoading] = useState(false);
  const [jobFormOptionsLoading, setJobFormOptionsLoading] = useState(false);
  const [jobModalError, setJobModalError] = useState<string | null>(null);

  // Input method for the scan card: keyboard (default), camera, or nfc
  const [inputMethod, setInputMethod] = useState<InputMethod>('keyboard');

  // Stable ref so camera/NFC callbacks can always reach the latest scan handler
  const processCodeRef = useRef<(code: string) => void>(() => {});

  // Ref for result auto-dismiss timeout – prevents an older timeout from
  // clearing a newer result when scans happen in quick succession.
  const resultTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Re-entrancy guard – prevents overlapping async scan requests
  const inFlightRef = useRef(false);

  const scheduleScanResultDismiss = useCallback(() => {
    if (resultTimeoutRef.current !== null) {
      clearTimeout(resultTimeoutRef.current);
    }
    resultTimeoutRef.current = setTimeout(() => {
      setScanResult(null);
      resultTimeoutRef.current = null;
    }, 3000);
  }, []);

  // LED state
  const [ledActive, setLedActive] = useState(false);
  const [ledStatus, setLedStatus] = useState<LEDStatus | null>(null);
  const [ledLoading, setLedLoading] = useState(false);

  const loadJobDetails = useCallback(async (jobId: number, options: { highlight?: boolean } = {}) => {
    try {
      setLoading(true);
      const { data } = await jobsApi.getById(jobId, { source: 'local' });
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
  }, []);

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
  }, [urlJobId, loadJobDetails]);

  const loadLEDStatus = async () => {
    try {
      const { data } = await ledApi.getStatus();
      setLedStatus(data);
    } catch (error) {
      console.error('Failed to load LED status:', error);
    }
  };

  const applyLocalScanResult = useCallback((action: 'outtake' | 'intake', device?: Record<string, any> | null) => {
    const deviceId = String(device?.device_id ?? device?.deviceID ?? '').trim();
    if (!deviceId) {
      return;
    }

    const resolvedProductID = Number(device?.product_id ?? device?.productID ?? 0) || 0;
    const resolvedProductName = String(device?.product_name ?? device?.productName ?? '').trim();
    const resolvedZoneName = String(device?.zone_name ?? device?.zoneName ?? '').trim();
    const resolvedBarcode = typeof device?.barcode === 'string' ? device.barcode : undefined;
    const resolvedQRCode = typeof device?.qr_code === 'string'
      ? device.qr_code
      : (typeof device?.qrCode === 'string' ? device.qrCode : undefined);

    setSelectedJob(prev => {
      if (!prev) return prev;

      const deviceExists = prev.devices.some(d => d.device_id === deviceId);

      let nextDevices = prev.devices.map(d => {
        if (d.device_id !== deviceId) {
          return d;
        }

        return {
          ...d,
          status: action === 'outtake' ? 'on_job' : 'in_storage',
          scanned: action === 'outtake',
          pack_status: action === 'outtake' ? 'issued' : 'pending',
          zone_name: action === 'outtake' ? '' : (resolvedZoneName || d.zone_name),
          product_name: resolvedProductName || d.product_name,
          barcode: resolvedBarcode || d.barcode,
          qr_code: resolvedQRCode || d.qr_code,
        };
      });

      if (!deviceExists && action === 'outtake') {
        nextDevices = [
          {
            device_id: deviceId,
            status: 'on_job',
            product_name: resolvedProductName || `Product ${resolvedProductID || '?'}`,
            zone_name: '',
            barcode: resolvedBarcode,
            qr_code: resolvedQRCode,
            pack_status: 'issued',
            scanned: true,
          },
          ...nextDevices,
        ];
      }

      const nextRequirements = (prev.product_requirements || []).map(req => {
        if (!resolvedProductID || req.product_id !== resolvedProductID) {
          return req;
        }

        const nextAssigned = action === 'outtake'
          ? Math.min(req.required, req.assigned + 1)
          : Math.max(0, req.assigned - 1);

        return {
          ...req,
          assigned: nextAssigned,
        };
      });

      return {
        ...prev,
        devices: nextDevices,
        product_requirements: nextRequirements,
      };
    });
  }, []);

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
      const { data } = await jobsApi.getAll({ status: 'open', source: 'local' });
      setJobs(data);
    } catch (error) {
      console.error('Failed to load jobs:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadRequirementTree = useCallback(async () => {
    try {
      setRequirementTreeLoading(true);
      const { data } = await devicesApi.getTree();
      setRequirementTreeData(data.treeData || []);
    } catch (error) {
      console.error('Failed to load requirement tree:', error);
      setRequirementTreeData([]);
    } finally {
      setRequirementTreeLoading(false);
    }
  }, []);

  const buildRequirementDrafts = useCallback((requirements: ProductRequirement[] | undefined) => {
    return (requirements || [])
      .map((req) => ({
        product_id: req.product_id,
        product_name: req.product_name,
        quantity: req.required,
        assigned: req.assigned,
      }))
      .sort((a, b) => a.product_name.localeCompare(b.product_name));
  }, []);

  const requirementServerSignature = selectedJob
    ? buildRequirementDrafts(selectedJob.product_requirements)
      .map((req) => `${req.product_id}:${req.quantity}:${req.product_name}`)
      .join('|')
    : '';

  const loadJobFormOptions = useCallback(async () => {
    try {
      setJobFormOptionsLoading(true);
      const { data } = await jobsApi.getFormOptions();
      setJobCustomers(data.customers || []);
      setJobStatuses(data.statuses || []);
      setDefaultJobStatusID(data.default_status_id);
      return data.default_status_id;
    } catch (error) {
      console.error('Failed to load job form options:', error);
      setJobCustomers([]);
      setJobStatuses([]);
      setDefaultJobStatusID(undefined);
      return undefined;
    } finally {
      setJobFormOptionsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (selectedJob) {
      loadRequirementTree();
    }
  }, [selectedJob, loadRequirementTree]);

  useEffect(() => {
    if (!selectedJob) {
      setRequirementDrafts([]);
      setRequirementTreeSearch('');
      setRequirementTreeExpanded(new Set());
      return;
    }
    setRequirementDrafts(buildRequirementDrafts(selectedJob.product_requirements));
    setRequirementTreeSearch('');
    setRequirementTreeExpanded(new Set());
  }, [selectedJob?.job_id, requirementServerSignature, buildRequirementDrafts]);

  useEffect(() => {
    loadJobFormOptions();
  }, [loadJobFormOptions]);

  const handleRequirementSelectionChange = ({
    product_id,
    product_name,
    quantity,
  }: {
    product_id: number;
    product_name: string;
    quantity: number;
  }) => {
    setRequirementDrafts((prev) => {
      const existing = prev.find((req) => req.product_id === product_id);
      const assigned = existing?.assigned
        ?? selectedJob?.product_requirements.find((req) => req.product_id === product_id)?.assigned
        ?? 0;
      const next = prev.filter((req) => req.product_id !== product_id);
      if (quantity > 0) {
        next.push({ product_id, product_name, quantity, assigned });
      }
      return next.sort((left, right) => left.product_name.localeCompare(right.product_name));
    });
    setRequirementMessage(null);
  };

  const handleRequirementExpandAll = (nodeIds: string[]) => {
    setRequirementTreeExpanded(new Set(nodeIds));
  };

  const handleRequirementCollapseAll = () => {
    setRequirementTreeExpanded(new Set());
  };

  const handleRequirementToggleNode = (nodeId: string) => {
    setRequirementTreeExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  };

	const handleSaveRequirements = async () => {
		if (!selectedJob) return;
		setSavingRequirement(true);
		setRequirementMessage(null);
		try {
			await jobsApi.replaceRequirements(selectedJob.job_id, {
				requirements: requirementDrafts
					.filter((req) => req.quantity > 0)
					.map((req) => ({ product_id: req.product_id, quantity: req.quantity })),
			});
			setRequirementMessage({
				type: 'success',
				text: `Saved ${requirementDrafts.length} product requirement${requirementDrafts.length === 1 ? '' : 's'}.`,
			});
			await loadJobDetails(selectedJob.job_id, { highlight: false });
		} catch (error: any) {
			setRequirementMessage({
				type: 'error',
				text: error?.response?.data?.error || t('jobsPage.requirementSaveError'),
			});
		} finally {
			setSavingRequirement(false);
		}
	};

  const handleCodeDetected = useCallback((code: string) => {
    processCodeRef.current(code);
  }, []);

  const barcodeScanner = useBarcodeScanner({ onDetected: handleCodeDetected });
  const nfcScanner = useNFCScanner({ onDetected: handleCodeDetected });

  const {
    startScanning: startBarcodeScanning,
    stopScanning: stopBarcodeScanning,
  } = barcodeScanner;
  const {
    startScanning: startNFCScanning,
    stopScanning: stopNFCScanning,
  } = nfcScanner;

  const handleInputMethodChange = useCallback((method: InputMethod) => {
    setScanCode('');
    setScanResult(null);
    setScanLoading(false);
    setInputMethod(method);
  }, []);

  // Start/stop scanners when input method changes
  useEffect(() => {
    let active = true;

    if (inputMethod !== 'camera') stopBarcodeScanning();
    if (inputMethod !== 'nfc') stopNFCScanning();

    if (inputMethod === 'camera') {
      Promise.resolve(startBarcodeScanning())
        .then(() => {
          if (!active) {
            stopBarcodeScanning();
          }
        })
        .catch((error) => {
          console.error('Failed to start barcode scanner:', error);
        });
    } else if (inputMethod === 'nfc') {
      Promise.resolve(startNFCScanning())
        .then(() => {
          if (!active) {
            stopNFCScanning();
          }
        })
        .catch((error) => {
          console.error('Failed to start NFC scanner:', error);
        });
    }

    return () => {
      active = false;
      stopBarcodeScanning();
      stopNFCScanning();
    };
  }, [inputMethod, startBarcodeScanning, stopBarcodeScanning, startNFCScanning, stopNFCScanning]);

  // Reset input method and stop scanners when leaving job detail view
  useEffect(() => {
    if (!selectedJob) {
      stopBarcodeScanning();
      stopNFCScanning();
      setInputMethod('keyboard');
    }
  }, [selectedJob, stopBarcodeScanning, stopNFCScanning]);

  // Stop scanners and clear pending result timeout on unmount
  useEffect(() => {
    return () => {
      stopBarcodeScanning();
      stopNFCScanning();
      if (resultTimeoutRef.current !== null) {
        clearTimeout(resultTimeoutRef.current);
      }
    };
  }, [stopBarcodeScanning, stopNFCScanning]);

  // Keep ref pointing to the latest processCode so camera/NFC callbacks are never stale
  const processCode = useCallback(async (code: string) => {
    // Re-entrancy guard: ignore new detections while a scan is already in-flight
    if (inFlightRef.current) return;

    const rawCode = code.trim();
    if (!rawCode) {
      return;
    }

    const normalizedCode = rawCode.toUpperCase();

    // Detect job code scans (e.g., JOB0001)
    if (JOB_CODE_PATTERN.test(normalizedCode)) {
      inFlightRef.current = true;
      setScanLoading(true);
      setScanResult(null);

      try {
        const numericPart = parseInt(normalizedCode.replace(/\D/g, ''), 10);
        if (Number.isNaN(numericPart)) {
          throw new Error(t('jobsPage.invalidJobId'));
        }

        await loadJobDetails(numericPart, { highlight: true });
        navigate(`/jobs/${numericPart}`);
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
        inFlightRef.current = false;
        scheduleScanResultDismiss();
      }

      return;
    }

    if (!selectedJob) {
      setScanResult({
        success: false,
        message: t('jobsPage.selectJobFirst'),
      });
      setScanLoading(false);
      scheduleScanResultDismiss();
      return;
    }

    inFlightRef.current = true;
    setScanLoading(true);
    setScanResult(null);

    try {
      // Process scan with job context for outtake; intake returns devices from job.
      const { data } = await scansApi.process({
        scan_code: rawCode,
        action: scanAction,
        job_id: scanAction === 'outtake' ? selectedJob.job_id : undefined,
      });

      setScanResult({
        success: data.success,
        message: data.message,
      });

      setScanCode('');

      if (data.success) {
        applyLocalScanResult(scanAction, data.device || null);
      }
    } catch (error: any) {
      console.error('Scan failed:', error);
      setScanResult({
        success: false,
        message: error.response?.data?.error || t('scan.scanError'),
      });
    } finally {
      setScanLoading(false);
      inFlightRef.current = false;
      // Clear result after 3 seconds
      scheduleScanResultDismiss();
    }
  }, [t, navigate, scanAction, selectedJob, loadJobDetails, applyLocalScanResult, scheduleScanResultDismiss]);

  // Keep submitCodeRef in sync with the latest processCode so scanner callbacks
  // (which are memoised on mount) can always reach the current state closure.
  useEffect(() => {
    processCodeRef.current = processCode;
  }, [processCode]);

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
    setScanAction('outtake');
    navigate('/jobs');
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

  const openCreateModal = () => {
    setEditingJob(null);
    setJobFormData({
      job_code: '',
      description: '',
      start_date: '',
      end_date: '',
      status_id: defaultJobStatusID,
      customer_id: undefined,
    });
    setJobModalError(null);
    setJobModalOpen(true);
    if (jobCustomers.length === 0 && jobStatuses.length === 0) {
      void loadJobFormOptions();
    }
  };

  const openEditModal = (job: Job) => {
    setEditingJob(job);
    setJobFormData({
      job_code: job.job_code,
      description: job.description || '',
      start_date: job.start_date ? job.start_date.slice(0, 10) : '',
      end_date: job.end_date ? job.end_date.slice(0, 10) : '',
      customer_id: job.customer_id,
      status_id: job.status_id,
    });
    setJobModalError(null);
    setJobModalOpen(true);
    if (jobCustomers.length === 0 && jobStatuses.length === 0) {
      void loadJobFormOptions();
    }
  };

  const handleStartDateChange = (value: string) => {
    setJobFormData(prev => ({
      ...prev,
      start_date: value,
      end_date: !prev.end_date && value ? value : prev.end_date,
    }));
  };

  const handleJobModalSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!jobFormData.job_code.trim()) {
      setJobModalError('Job code is required');
      return;
    }
    setJobModalLoading(true);
    setJobModalError(null);
    try {
      const payload: JobCreateInput = {
        job_code: jobFormData.job_code.trim(),
        description: jobFormData.description || undefined,
        start_date: jobFormData.start_date || undefined,
        end_date: jobFormData.end_date || undefined,
        customer_id: jobFormData.customer_id && jobFormData.customer_id > 0 ? jobFormData.customer_id : undefined,
        status_id: jobFormData.status_id && jobFormData.status_id > 0 ? jobFormData.status_id : undefined,
      };
      if (editingJob) {
        await jobsApi.update(editingJob.job_id, payload);
      } else {
        await jobsApi.create(payload);
      }
      setJobModalOpen(false);
      loadJobs();
    } catch (err: any) {
      setJobModalError(err?.response?.data?.error || 'Failed to save job');
    } finally {
      setJobModalLoading(false);
    }
  };

  const handleDeleteJob = async (job: Job) => {
    if (!confirm(`Delete job ${job.job_code}? This cannot be undone.`)) return;
    try {
      await jobsApi.delete(job.job_id);
      loadJobs();
    } catch (err: any) {
      alert(err?.response?.data?.error || 'Failed to delete job');
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
          <div className="mb-8 flex items-start justify-between gap-4">
            <div>
              <h1 className="text-4xl font-bold text-white mb-2">{t('jobsPage.openJobsTitle')}</h1>
              <p className="text-gray-400">{t('jobsPage.openJobsSubtitle')}</p>
            </div>
            <button
              onClick={openCreateModal}
              className="flex items-center gap-2 px-4 py-2 bg-accent-red hover:bg-red-700 text-white font-semibold rounded-xl transition-all whitespace-nowrap"
            >
              <PlusCircle className="w-5 h-5" />
              New Job
            </button>
          </div>

          {loading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red"></div>
              <p className="text-gray-400 mt-4">{t('jobsPage.loadingJobs')}</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {jobs.length === 0 && (
                <div className="col-span-full glass-dark rounded-2xl p-12 text-center">
                  <Package className="w-16 h-16 text-gray-600 mx-auto mb-4" />
                  <p className="text-gray-400 text-lg">{t('jobsPage.noOpenJobs')}</p>
                </div>
              )}
              {jobs.map((job) => (
                <div
                  key={job.job_id}
                  className="glass-dark rounded-2xl p-6 border-2 border-white/10 hover:border-accent-red transition-all text-left group relative"
                >
                  {/* Edit / Delete actions */}
                  <div className="absolute top-4 right-4 flex gap-1">
                    <button
                      onClick={(e) => { e.stopPropagation(); openEditModal(job); }}
                      className="p-1.5 rounded-lg text-gray-400 hover:text-white hover:bg-white/10 transition-all"
                      title="Edit job"
                    >
                      <Pencil className="w-4 h-4" />
                    </button>
                    <button
                      onClick={(e) => { e.stopPropagation(); handleDeleteJob(job); }}
                      className="p-1.5 rounded-lg text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-all"
                      title="Delete job"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>

                  <button
                    className="w-full text-left"
                    onClick={() => navigate(`/jobs/${job.job_id}`)}
                  >
                    <div className="flex items-start justify-between mb-4 pr-16">
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
                          <span>{formatDateISO(job.start_date)}</span>
                        </div>
                      )}

                      <div className="flex items-center gap-2 text-accent-red font-semibold">
                        <Package className="w-4 h-4" />
                        <span>{t('jobsPage.deviceCount', { count: job.device_count })}</span>
                      </div>
                      {job.requirements_count > 0 && (
                        <div className="flex items-center gap-2 text-yellow-400 font-semibold">
                          <ClipboardList className="w-4 h-4" />
                          <span>{t('jobsPage.requirementsCount', { count: job.requirements_count })}</span>
                        </div>
                      )}
                    </div>

                    <div className="mt-4 flex items-center gap-2 text-accent-red group-hover:gap-3 transition-all">
                      <span className="font-semibold">{t('jobsPage.select')}</span>
                      <ArrowRight className="w-4 h-4" />
                    </div>
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* Create / Edit Job Modal */}
          {jobModalOpen && (
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
                <div className="glass-dark rounded-2xl p-6 w-full max-w-md border border-white/10 shadow-2xl">
                  <div className="flex items-center justify-between mb-6">
                    <h2 className="text-xl font-bold text-white">
                      {editingJob ? 'Edit Job' : 'New Job'}
                    </h2>
                    <button onClick={() => setJobModalOpen(false)} className="text-gray-400 hover:text-white">
                      <X className="w-5 h-5" />
                    </button>
                  </div>

                  <form onSubmit={handleJobModalSubmit} className="space-y-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-300 mb-1">Job Code *</label>
                      <input
                        type="text"
                        value={jobFormData.job_code}
                        onChange={(e) => setJobFormData(prev => ({ ...prev, job_code: e.target.value }))}
                        className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-accent-red"
                        placeholder="JOB001"
                        required
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-300 mb-1">Description</label>
                      <textarea
                        value={jobFormData.description || ''}
                        onChange={(e) => setJobFormData(prev => ({ ...prev, description: e.target.value }))}
                        className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-accent-red resize-none"
                        placeholder="Optional description"
                        rows={3}
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">Start Date</label>
                        <input
                          type="date"
                          value={jobFormData.start_date || ''}
                          onFocus={(e) => {
                            if (typeof (e.currentTarget as HTMLInputElement & { showPicker?: () => void }).showPicker === 'function') {
                              (e.currentTarget as HTMLInputElement & { showPicker: () => void }).showPicker();
                            }
                          }}
                          onChange={(e) => handleStartDateChange(e.target.value)}
                          className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-2.5 text-white focus:outline-none focus:border-accent-red"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">End Date</label>
                        <input
                          type="date"
                          value={jobFormData.end_date || ''}
                          onFocus={(e) => {
                            if (typeof (e.currentTarget as HTMLInputElement & { showPicker?: () => void }).showPicker === 'function') {
                              (e.currentTarget as HTMLInputElement & { showPicker: () => void }).showPicker();
                            }
                          }}
                          onChange={(e) => setJobFormData(prev => ({ ...prev, end_date: e.target.value }))}
                          className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-2.5 text-white focus:outline-none focus:border-accent-red"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">Customer</label>
                        <select
                          value={jobFormData.customer_id || 0}
                          onChange={(e) => setJobFormData(prev => ({ ...prev, customer_id: Number(e.target.value) || undefined }))}
                          className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-2.5 text-white focus:outline-none focus:border-accent-red"
                        >
                          <option value={0}>No customer</option>
                          {jobCustomers.map((customer) => (
                            <option key={customer.customer_id} value={customer.customer_id}>
                              {customer.display_name}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">Status</label>
                        <select
                          value={jobFormData.status_id || 0}
                          onChange={(e) => setJobFormData(prev => ({ ...prev, status_id: Number(e.target.value) || undefined }))}
                          className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-2.5 text-white focus:outline-none focus:border-accent-red"
                        >
                          <option value={0}>Default</option>
                          {jobStatuses.map((status) => (
                            <option key={status.status_id} value={status.status_id}>
                              {status.name}
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>

                    {jobFormOptionsLoading && (
                      <p className="text-xs text-gray-500">Loading customer and status options...</p>
                    )}

                    {jobModalError && (
                      <p className="text-red-400 text-sm">{jobModalError}</p>
                    )}

                    <div className="flex gap-3 pt-2">
                      <button
                        type="button"
                        onClick={() => setJobModalOpen(false)}
                        className="flex-1 px-4 py-2.5 rounded-xl border border-white/10 text-gray-300 hover:bg-white/5 transition-all"
                      >
                        Cancel
                      </button>
                      <button
                        type="submit"
                        disabled={jobModalLoading}
                        className="flex-1 px-4 py-2.5 rounded-xl bg-accent-red hover:bg-red-700 text-white font-semibold transition-all disabled:opacity-50"
                      >
                        {jobModalLoading ? 'Saving\u2026' : (editingJob ? 'Update' : 'Create')}
                      </button>
                    </div>
                  </form>
                </div>
              </div>
          )}
        </div>
      </div>
    );
  }

  // Job Details & Scan View
  // Prefer product requirement based progress when available, otherwise fall
  // back to device-based progress for legacy jobs.
  const getRequirementStats = (reqs: ProductRequirement[] | undefined, devices: JobDevice[]) => {
    if (reqs && reqs.length > 0) {
      const total = reqs.reduce((acc, r) => acc + (r.required || 0), 0);
      const scanned = reqs.reduce(
        (acc, r) => acc + Math.min(r.assigned || 0, r.required || 0),
        0
      );
      const remaining = reqs.reduce(
        (acc, r) => acc + Math.max(0, (r.required || 0) - (r.assigned || 0)),
        0
      );
      return { total, scanned, remaining };
    }
    return getDeviceStats(devices);
  };

  const stats = getRequirementStats(selectedJob.product_requirements, selectedJob.devices);
  const progress = stats.total > 0 ? Math.min((stats.scanned / stats.total) * 100, 100) : 0;

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
                      {formatDateISO(selectedJob.start_date)}
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
          {/* Product Requirements */}
          {selectedJob.product_requirements && (
            <div className="lg:col-span-2 glass-dark rounded-2xl p-6 border-2 border-white/10">
              <h2 className="text-2xl font-bold text-white mb-4 flex items-center gap-2">
                <ClipboardList className="w-6 h-6" />
                {t('jobsPage.productRequirements')}
              </h2>

              <JobRequirementTree
                treeData={requirementTreeData}
                loading={requirementTreeLoading}
                search={requirementTreeSearch}
                onSearchChange={setRequirementTreeSearch}
                expandedNodes={requirementTreeExpanded}
                onToggleNode={handleRequirementToggleNode}
                onExpandAll={handleRequirementExpandAll}
                onCollapseAll={handleRequirementCollapseAll}
                selections={requirementDrafts}
                onSetSelection={handleRequirementSelectionChange}
              />

              {requirementMessage && (
                <p className={`mt-3 text-sm ${requirementMessage.type === 'success' ? 'text-green-400' : 'text-red-400'}`}>
                  {requirementMessage.text}
                </p>
              )}

              <div className="mt-4 flex justify-end">
                <button
                  type="button"
                  onClick={handleSaveRequirements}
                  disabled={savingRequirement || !selectedJob}
                  className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-accent-red text-white font-semibold disabled:opacity-60"
                >
                  {savingRequirement ? 'Saving...' : 'Save Requirements'}
                </button>
              </div>
            </div>
          )}

          {/* Scan Interface */}
          <div className="glass-dark rounded-2xl p-6 border-2 border-white/10">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-2xl font-bold text-white">
                {scanAction === 'outtake'
                  ? t('jobsPage.outtakeDevice')
                  : t('jobsPage.intakeDevice')}
              </h2>
            </div>

            <div role="group" aria-label={t('scan.actions')} className="flex gap-2 mb-4">
              <button
                type="button"
                onClick={() => setScanAction('outtake')}
                className={`flex-1 py-2 rounded-lg text-sm font-semibold transition-all ${
                  scanAction === 'outtake' ? 'bg-accent-red text-white' : 'bg-white/5 text-gray-300 hover:text-white'
                }`}
              >
                {t('scan.actions.outtake')}
              </button>
              <button
                type="button"
                onClick={() => setScanAction('intake')}
                className={`flex-1 py-2 rounded-lg text-sm font-semibold transition-all ${
                  scanAction === 'intake' ? 'bg-emerald-600 text-white' : 'bg-white/5 text-gray-300 hover:text-white'
                }`}
              >
                {t('scan.actions.intake')}
              </button>
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

            <div role="group" aria-label={t('scan.inputMethods.label')} className="flex gap-1 p-1 bg-white/5 rounded-xl mb-4">
              <button
                type="button"
                onClick={() => handleInputMethodChange('keyboard')}
                aria-pressed={inputMethod === 'keyboard'}
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
                  aria-pressed={inputMethod === 'camera'}
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
                  aria-pressed={inputMethod === 'nfc'}
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
                {scanLoading
                  ? t('jobsPage.scanning')
                  : (scanAction === 'outtake' ? t('jobsPage.outtakeDevice') : t('jobsPage.intakeDevice'))}
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
