import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function getStatusColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'in_storage':
      return 'text-gray-400';
    case 'on_job':
    case 'rented':
      return 'text-accent-red';
    case 'defective':
      return 'text-yellow-500';
    case 'repair':
      return 'text-blue-400';
    case 'free':
      return 'text-green-500';
    default:
      return 'text-gray-500';
  }
}

export function formatStatus(status: string): string {
  return status.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase());
}

export function formatDateISO(dateStr?: string | null): string {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  if (Number.isNaN(d.getTime())) return '';
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export function formatLocalDateTime(dateStr?: string | null): string {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  if (Number.isNaN(d.getTime())) return '';
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  const hours = String(d.getHours()).padStart(2, '0');
  const minutes = String(d.getMinutes()).padStart(2, '0');
  return `${year}-${month}-${day} ${hours}:${minutes}`;
}
