import axios from 'axios';

// Use relative path so it works on any host/platform
const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Types
export interface Device {
  device_id: string;
  product_id?: number;
  product_name?: string;
  barcode?: string;
  qr_code?: string;
  serial_number?: string;
  status: string;
  current_location?: string;
  zone_id?: number;
  zone_name?: string;
  zone_code?: string;
  case_name?: string;
  job_number?: string;
  condition_rating?: number;
  usage_hours?: number;
}

export interface DeviceTreeDevice {
  device_id: string;
  product_name: string;
  status: string;
  barcode?: string;
  serial_number?: string;
  zone_id?: number;
  zone_code?: string;
}

export interface DeviceTreeSubbiercategory {
  id: string | number;
  name: string;
  devices: DeviceTreeDevice[];
  device_count: number;
}

export interface DeviceTreeSubcategory {
  id: string | number;
  name: string;
  subbiercategories: DeviceTreeSubbiercategory[];
  direct_devices: DeviceTreeDevice[];
  device_count: number;
}

export interface DeviceTreeCategory {
  id: number;
  name: string;
  subcategories: DeviceTreeSubcategory[];
  direct_devices: DeviceTreeDevice[];
  device_count: number;
}

export interface DeviceTreeResponse {
  treeData: DeviceTreeCategory[];
}

export interface Zone {
  zone_id: number;
  code: string;
  name: string;
  type: string;
  description?: string | null;
  parent_zone_id?: number | null;
  capacity?: number | null;
  is_active: boolean;
}

export interface ZoneTypeDefinition {
  id: number;
  key: string;
  label: string;
  description?: string | null;
  default_led_pattern?: string;
  default_led_color?: string;
  default_intensity?: number;
}

export interface ScanRequest {
  scan_code: string;
  action: 'intake' | 'outtake' | 'check' | 'transfer';
  job_id?: number;
  zone_id?: number;
  notes?: string;
}

export interface ScanResponse {
  success: boolean;
  message: string;
  device?: Device;
  action: string;
  previous_status?: string;
  new_status?: string;
  duplicate: boolean;
}

export interface DashboardStats {
  in_storage: number;
  on_job: number;
  defective: number;
  total: number;
}

export interface Movement {
  movement_id: number;
  device_id: string;
  action: 'intake' | 'outtake' | 'transfer' | 'return' | 'move' | string;
  timestamp: string;
  from_zone_id?: number;
  to_zone_id?: number;
  from_job_id?: number;
  to_job_id?: number;
  barcode?: string;
  serial_number?: string;
  product_name?: string;
  from_zone_name?: string;
  to_zone_name?: string;
  from_job_description?: string;
  to_job_description?: string;
  performed_by?: string;
}

export interface Job {
  job_id: number;
  job_code: string;
  description?: string;
  start_date?: string;
  end_date?: string;
  status: string;
  customer_first_name?: string;
  customer_last_name?: string;
  device_count: number;
}

export interface JobDevice {
  device_id: string;
  status: string;
  product_name: string;
  zone_name?: string;
  barcode?: string;
  qr_code?: string;
  pack_status: string;
  scanned: boolean;
}

export interface JobSummary {
  job_id: number;
  job_code: string;
  description?: string;
  start_date?: string;
  end_date?: string;
  status: string;
  customer_first_name?: string;
  customer_last_name?: string;
  devices: JobDevice[];
}

// API Functions
export const dashboardApi = {
  getStats: () => api.get<DashboardStats>('/dashboard/stats'),
  getRecentMovements: (limit: number = 10) =>
    api.get<Movement[]>('/movements', { params: { limit } }),
};

export const devicesApi = {
  getAll: (params?: { status?: string; zone_id?: number; limit?: number }) =>
    api.get<Device[]>('/devices', { params }),
  getById: (id: string) => api.get<Device>(`/devices/${id}`),
  getMovements: (id: string) => api.get(`/devices/${id}/movements`),
  getTree: () => api.get<DeviceTreeResponse>('/devices/tree'),
};

export const zonesApi = {
  getAll: () => api.get<Zone[]>('/zones'),
  getById: (id: number) => api.get<Zone>(`/zones/${id}`),
  getByScan: (scanCode: string) => api.get<Zone>('/zones/scan', { params: { scan_code: scanCode } }),
  create: (data: Partial<Zone>) => api.post<Zone>('/zones', data),
  update: (id: number, data: Partial<Zone>) => api.put(`/zones/${id}`, data),
  delete: (id: number) => api.delete(`/zones/${id}`),
};

export const zoneTypesApi = {
  getAll: () => api.get<ZoneTypeDefinition[]>('/zone-types'),
};

export const scansApi = {
  process: (data: ScanRequest) => api.post<ScanResponse>('/scans', data),
  getHistory: (limit: number = 50) => api.get(`/scans/history`, { params: { limit } }),
};

export const jobsApi = {
  getAll: (params?: { status?: string }) => api.get<Job[]>('/jobs', { params }),
  getById: (id: number) => api.get<JobSummary>(`/jobs/${id}`),
};

export interface Defect {
  defect_id: number;
  device_id: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'open' | 'in_progress' | 'repaired' | 'closed';
  title: string;
  description: string;
  reported_at: string;
  repair_cost?: number;
  repaired_at?: string;
  closed_at?: string;
  product_name?: string;
}

export interface Inspection {
  schedule_id: number;
  device_id?: string;
  product_id?: number;
  inspection_type: string;
  interval_days: number;
  last_inspection?: string;
  next_inspection?: string;
  is_active: boolean;
  product_name?: string;
  device_name?: string;
}

export interface MaintenanceStats {
  open_defects: number;
  in_progress_defects: number;
  repaired_defects: number;
  overdue_inspections: number;
  upcoming_inspections: number;
}

export const maintenanceApi = {
  getStats: () => api.get<MaintenanceStats>('/maintenance/stats'),
  getDefects: (params?: { status?: string; severity?: string }) =>
    api.get<Defect[]>('/defects', { params }),
  createDefect: (data: {
    device_id: string;
    severity: string;
    title: string;
    description: string;
  }) => api.post<{ defect_id: number; message: string }>('/defects', data),
  updateDefect: (id: number, data: {
    status?: string;
    repair_cost?: number;
    repair_notes?: string;
  }) => api.put(`/defects/${id}`, data),
  getInspections: (params?: { status?: string }) =>
    api.get<Inspection[]>('/maintenance/inspections', { params }),
};

// LED Control Types
export interface LEDStatus {
  mqtt_connected: boolean;
  mqtt_dry_run: boolean;
  mapping_loaded: boolean;
  warehouse_id: string;
  total_shelves: number;
  total_bins: number;
}

export interface LEDMapping {
  warehouse_id: string;
  shelves: Array<{
    shelf_id: string;
    bins: Array<{
      bin_id: string;
      pixels: number[];
    }>;
  }>;
  led_strip: {
    length: number;
    data_pin: number;
    chipset: string;
  };
  defaults: {
    color: string;
    pattern: string;
    intensity: number;
    speed?: number;
  };
}

export interface LEDAppearance {
  color: string;
  pattern: string;
  intensity: number;
  speed: number;
}

export interface LEDJobHighlightSettings {
  mode: 'all_bins' | 'required_only';
  required: LEDAppearance;
  non_required: LEDAppearance;
}

export interface LEDController {
  id: number;
  controller_id: string;
  display_name: string;
  topic_suffix: string;
  is_active: boolean;
  last_seen?: string | null;
  metadata?: Record<string, unknown> | null;
  ip_address?: string | null;
  hostname?: string | null;
  firmware_version?: string | null;
  mac_address?: string | null;
  status_data?: Record<string, unknown> | null;
  zone_types?: ZoneTypeDefinition[];
}

export interface LEDControllerPayload {
  controller_id?: string;
  display_name?: string;
  topic_suffix?: string;
  is_active?: boolean;
  metadata?: Record<string, unknown> | null;
  zone_type_ids?: number[];
}

export const ledApi = {
  getStatus: () => api.get<LEDStatus>('/led/status'),
  highlightJob: (jobId: number) => api.post(`/led/highlight?job_id=${jobId}`),
  clear: () => api.post('/led/clear'),
  identify: () => api.post('/led/identify'),
  testBin: (shelfId: string, binId: string) =>
    api.post(`/led/test?shelf_id=${shelfId}&bin_id=${binId}`),
  locateBin: (binCode: string) =>
    api.post(`/led/locate?bin_code=${binCode}`),
  getJobSettings: () => api.get<LEDJobHighlightSettings>('/admin/led/job-highlights'),
  updateJobSettings: (settings: LEDJobHighlightSettings) => api.put('/admin/led/job-highlights', settings),
  getMapping: () => api.get<LEDMapping>('/admin/led/mapping'),
  updateMapping: (mapping: LEDMapping) => api.put('/admin/led/mapping', mapping),
  validateMapping: (mapping: LEDMapping) => api.post('/admin/led/mapping/validate', mapping),
  preview: (appearances: LEDAppearance[], clearBefore: boolean = false, targetBinId?: string) => {
    const payload: Record<string, unknown> = {
      appearances,
    };
    if (clearBefore) {
      payload.clear_before = true;
    }
    if (targetBinId && targetBinId.trim().length > 0) {
      payload.target_bin_id = targetBinId.trim();
    }
    return api.post('/admin/led/preview', payload);
  },
  getControllers: () => api.get<LEDController[]>('/admin/led/controllers'),
  createController: (payload: LEDControllerPayload) => api.post('/admin/led/controllers', payload),
  updateController: (id: number, payload: LEDControllerPayload) => api.put(`/admin/led/controllers/${id}`, payload),
  deleteController: (id: number) => api.delete(`/admin/led/controllers/${id}`),
};
