import { get, post } from '@/utils/request'

// 土壤养护单（与后端 MaintenanceOutDTO 对应）
export interface MaintenanceOrder {
  id: number
  plot_id: number
  plot_code: string
  plot_name: string
  operator_id: number
  operator_name: string
  sampled_at: string | null
  ph_value: number
  fertility_issue: string
  suggestion: string
  status: MaintenanceStatusString
  actual_measures: string
  completed_at: string | null
  cancel_reason: string
  cancelled_at: string | null
  created_at: string
}

export type MaintenanceStatusString = 'pending' | 'processing' | 'completed' | 'cancelled'

export interface CreateMaintenancePayload {
  plot_id: number
  sampled_at: string
  ph_value: number
  fertility_issue: string
  suggestion: string
}

export interface CompleteMaintenancePayload {
  actual_measures: string
  completed_at: string
}

export interface MaintenanceListParams {
  page?: number
  page_size?: number
  plot_id?: number
  status?: string
}

export function listMaintenanceOrders(params?: MaintenanceListParams): Promise<{ list: MaintenanceOrder[]; total: number; page: number; page_size: number }> {
  return get('/maintenance-orders', { params })
}

export function getMaintenanceOrder(id: number): Promise<MaintenanceOrder> {
  return get(`/maintenance-orders/${id}`)
}

export function createMaintenanceOrder(payload: CreateMaintenancePayload): Promise<MaintenanceOrder> {
  return post('/maintenance-orders', payload)
}

export function startMaintenanceOrder(id: number): Promise<MaintenanceOrder> {
  return post(`/maintenance-orders/${id}/start`)
}

export function completeMaintenanceOrder(id: number, payload: CompleteMaintenancePayload): Promise<MaintenanceOrder> {
  return post(`/maintenance-orders/${id}/complete`, payload)
}

export function cancelMaintenanceOrder(id: number, cancelReason: string): Promise<MaintenanceOrder> {
  return post(`/maintenance-orders/${id}/cancel`, { cancel_reason: cancelReason })
}

// 按地块查询养护历史
export function listMaintenanceHistoryByPlot(plotId: number): Promise<MaintenanceOrder[]> {
  return get(`/plots/${plotId}/maintenances`)
}
