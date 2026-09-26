import { get, post } from '@/utils/request'

export interface SoilCareOrder {
  id: number
  plot_id: number
  plot_code: string
  plot_name: string
  admin_id: number
  admin_name: string
  status: string
  sampled_date: string
  ph_value: number
  fertility_issue: string
  treatment_advice: string
  actual_measures: string
  completed_date: string | null
  cancel_reason: string
  created_at: string
}

export interface CreateCarePayload {
  plot_id: number
  sampled_date: string
  ph_value: number
  fertility_issue: string
  treatment_advice: string
}

export function listCareOrders(params?: Record<string, any>): Promise<{ list: SoilCareOrder[]; total: number; page: number; page_size: number }> {
  return get('/soil-care-orders', { params })
}

export function getCareOrder(id: number): Promise<SoilCareOrder> {
  return get(`/soil-care-orders/${id}`)
}

export function createCareOrder(payload: CreateCarePayload): Promise<SoilCareOrder> {
  return post('/soil-care-orders', payload)
}

export function startCareOrder(id: number): Promise<SoilCareOrder> {
  return post(`/soil-care-orders/${id}/start`)
}

export function completeCareOrder(id: number, payload: { actual_measures: string; completed_date?: string }): Promise<SoilCareOrder> {
  return post(`/soil-care-orders/${id}/complete`, payload)
}

export function cancelCareOrder(id: number, payload: { cancel_reason: string }): Promise<SoilCareOrder> {
  return post(`/soil-care-orders/${id}/cancel`, payload)
}

export function listAdoptedPlotsForCare(): Promise<Array<{ id: number; code: string; name: string }>> {
  return get('/care/plots/adopted')
}
