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
  status: string;
  current_location?: string;
  zone_id?: number;
  zone_name?: string;
  case_name?: string;
  job_number?: string;
  condition_rating: number;
  usage_hours: number;
}

export interface Zone {
  zone_id: number;
  code: string;
  name: string;
  type: string;
  description?: string;
  capacity?: number;
  is_active: boolean;
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

// API Functions
export const dashboardApi = {
  getStats: () => api.get<DashboardStats>('/dashboard/stats'),
};

export const devicesApi = {
  getAll: (params?: { status?: string; zone_id?: number; limit?: number }) =>
    api.get<Device[]>('/devices', { params }),
  getById: (id: string) => api.get<Device>(`/devices/${id}`),
  getMovements: (id: string) => api.get(`/devices/${id}/movements`),
};

export const zonesApi = {
  getAll: () => api.get<Zone[]>('/zones'),
  getById: (id: number) => api.get<Zone>(`/zones/${id}`),
  create: (data: Partial<Zone>) => api.post<Zone>('/zones', data),
  update: (id: number, data: Partial<Zone>) => api.put(`/zones/${id}`, data),
  delete: (id: number) => api.delete(`/zones/${id}`),
};

export const scansApi = {
  process: (data: ScanRequest) => api.post<ScanResponse>('/scans', data),
  getHistory: (limit: number = 50) => api.get(`/scans/history`, { params: { limit } }),
};
