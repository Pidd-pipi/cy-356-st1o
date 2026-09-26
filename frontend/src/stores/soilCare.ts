import { defineStore } from 'pinia'
import {
  cancelCareOrder,
  completeCareOrder,
  createCareOrder,
  listCareOrders,
  startCareOrder,
  type CreateCarePayload,
  type SoilCareOrder
} from '@/api/soilCare'

interface CareState {
  orders: SoilCareOrder[]
  total: number
  loading: boolean
}

export const useCareStore = defineStore('soilCare', {
  state: (): CareState => ({ orders: [], total: 0, loading: false }),
  actions: {
    async fetchOrders(params?: Record<string, any>) {
      this.loading = true
      try {
        const data = await listCareOrders(params)
        this.orders = data.list
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async create(payload: CreateCarePayload) {
      await createCareOrder(payload)
    },
    async start(id: number) {
      await startCareOrder(id)
    },
    async complete(id: number, payload: { actual_measures: string; completed_date?: string }) {
      await completeCareOrder(id, payload)
    },
    async cancel(id: number, cancelReason: string) {
      await cancelCareOrder(id, { cancel_reason: cancelReason })
    }
  }
})
