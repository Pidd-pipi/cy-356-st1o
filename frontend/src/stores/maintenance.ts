import { defineStore } from 'pinia'
import {
  cancelMaintenanceOrder,
  completeMaintenanceOrder,
  createMaintenanceOrder,
  listMaintenanceOrders,
  startMaintenanceOrder,
  type CreateMaintenancePayload,
  type CompleteMaintenancePayload,
  type MaintenanceListParams,
  type MaintenanceOrder
} from '@/api/maintenance'

interface MaintenanceState {
  orders: MaintenanceOrder[]
  total: number
  loading: boolean
}

export const useMaintenanceStore = defineStore('maintenance', {
  state: (): MaintenanceState => ({ orders: [], total: 0, loading: false }),
  actions: {
    async fetchOrders(params?: MaintenanceListParams) {
      this.loading = true
      try {
        const data = await listMaintenanceOrders(params)
        this.orders = data.list
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async create(payload: CreateMaintenancePayload) {
      await createMaintenanceOrder(payload)
      await this.fetchOrders({ page: 1, page_size: 10 })
    },
    async start(id: number) {
      await startMaintenanceOrder(id)
    },
    async complete(id: number, payload: CompleteMaintenancePayload) {
      await completeMaintenanceOrder(id, payload)
    },
    async cancel(id: number, reason: string) {
      await cancelMaintenanceOrder(id, reason)
    }
  }
})
