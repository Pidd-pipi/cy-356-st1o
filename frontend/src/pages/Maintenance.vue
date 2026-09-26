<template>
  <div class="page-card">
    <div style="display: flex; justify-content: space-between; align-items: center">
      <h3 class="page-title">土壤养护单</h3>
      <el-button v-if="isAdmin" type="primary" @click="openCreate">+ 登记养护单</el-button>
    </div>

    <el-alert
      v-if="!isAdmin"
      type="info"
      :closable="false"
      show-icon
      title="管理员会为您认养的地块登记土壤采样与养护处理；养护期间地块暂不能创建新的种植计划，养护完成后自动恢复。"
      style="margin-bottom: 16px"
    />

    <el-card shadow="never" style="margin-bottom: 16px">
      <el-form :inline="true" @submit.prevent>
        <el-form-item label="地块">
          <el-select v-model="filterPlotId" :placeholder="isAdmin ? '全部地块' : '请选择我的地块'" clearable filterable style="width: 240px" @change="reload">
            <el-option v-for="p in selectablePlots" :key="p.id" :label="`${p.code} ${p.name}`" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filterStatus" placeholder="全部状态" clearable style="width: 140px" @change="reload">
            <el-option v-for="(m, k) in MaintenanceStatusMeta" :key="k" :label="m.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <DataTable
      :data="store.orders"
      :loading="store.loading"
      :total="store.total"
      :page-size="pagination.size.value"
      :current-page="pagination.page.value"
      @update:current-page="onPage"
    >
      <el-table-column prop="plot_code" label="地块编号" width="100" />
      <el-table-column prop="plot_name" label="地块名称" min-width="140" />
      <el-table-column label="采样日期" width="110">
        <template #default="{ row }">{{ formatDate(row.sampled_at) }}</template>
      </el-table-column>
      <el-table-column label="pH 值" width="90">
        <template #default="{ row }">{{ Number(row.ph_value).toFixed(1) }}</template>
      </el-table-column>
      <el-table-column label="肥力问题" width="110">
        <template #default="{ row }">{{ FertilityIssueText[row.fertility_issue] || row.fertility_issue }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }"><StatusBadge :value="row.status" :meta-map="MaintenanceStatusMeta" /></template>
      </el-table-column>
      <el-table-column label="处理建议" min-width="200" show-overflow-tooltip prop="suggestion" />
      <el-table-column v-if="isAdmin" label="登记人" width="110">
        <template #default="{ row }">{{ row.operator_name || '-' }}</template>
      </el-table-column>
      <el-table-column label="操作" :width="isAdmin ? 260 : 110" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openDetail(row)">详情/历史</el-button>
          <template v-if="isAdmin">
            <el-button v-if="row.status === 'pending'" type="primary" size="small" @click="onStart(row)">开始处理</el-button>
            <el-button v-if="row.status === 'processing'" type="success" size="small" @click="openComplete(row)">完成</el-button>
            <el-button v-if="row.status === 'pending' || row.status === 'processing'" type="danger" size="small" @click="openCancel(row)">取消</el-button>
          </template>
        </template>
      </el-table-column>
    </DataTable>

    <!-- 登记养护单 -->
    <el-dialog v-model="createVisible" title="登记土壤养护单" width="560px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="已认养地块" required>
          <el-select v-model="createForm.plot_id" placeholder="选择已认养地块" filterable style="width: 100%">
            <el-option
              v-for="p in adoptedPlots"
              :key="p.id"
              :label="`${p.code} ${p.name}（认养人：${p.adopter?.nickname || p.adopter?.username || '-'}）`"
              :value="p.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="采样日期" required>
          <el-date-picker v-model="createForm.sampled_at" type="date" value-format="YYYY-MM-DD" :disabled-date="disableFuture" placeholder="选择土壤采样日期" style="width: 100%" />
        </el-form-item>
        <el-form-item label="pH 值" required>
          <el-input-number v-model="createForm.ph_value" :min="0" :max="14" :step="0.1" :precision="2" style="width: 200px" />
          <span class="muted" style="margin-left: 8px">适宜范围 6.0 ~ 7.5</span>
        </el-form-item>
        <el-form-item label="肥力问题" required>
          <el-select v-model="createForm.fertility_issue" style="width: 100%">
            <el-option v-for="(t, k) in FertilityIssueText" :key="k" :label="t" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="处理建议" required>
          <el-input v-model="createForm.suggestion" type="textarea" :rows="3" maxlength="512" show-word-limit placeholder="如：撒施生石灰调酸、增施腐熟有机肥、深翻静置一周" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="submitCreate">登记并开始养护</el-button>
      </template>
    </el-dialog>

    <!-- 完成养护单 -->
    <el-dialog v-model="completeVisible" title="完成土壤养护" width="520px">
      <el-form :model="completeForm" label-width="100px">
        <el-form-item label="实际措施" required>
          <el-input v-model="completeForm.actual_measures" type="textarea" :rows="4" maxlength="512" show-word-limit placeholder="填写实际采取的土壤改良措施" />
        </el-form-item>
        <el-form-item label="完成日期" required>
          <el-date-picker v-model="completeForm.completed_at" type="date" value-format="YYYY-MM-DD" :disabled-date="disableFuture" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="completeVisible = false">取消</el-button>
        <el-button type="success" :loading="acting" @click="submitComplete">确认完成，地块恢复可种植</el-button>
      </template>
    </el-dialog>

    <!-- 取消养护单 -->
    <el-dialog v-model="cancelVisible" title="取消土壤养护单" width="480px">
      <el-alert type="warning" :closable="false" show-icon title="取消后地块将立即恢复可种植；取消原因会永久保留在养护历史中。" style="margin-bottom: 12px" />
      <el-input v-model="cancelReason" type="textarea" :rows="4" maxlength="512" show-word-limit placeholder="请填写取消原因（必填），如：连续降雨无法施工，延期至下季度" />
      <template #footer>
        <el-button @click="cancelVisible = false">再想想</el-button>
        <el-button type="danger" :loading="acting" @click="submitCancel">确认取消</el-button>
      </template>
    </el-dialog>

    <!-- 详情 + 地块养护历史 -->
    <el-dialog v-model="detailVisible" title="养护单详情与地块养护历史" width="760px">
      <template v-if="detailOrder">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="地块">{{ detailOrder.plot_code }} {{ detailOrder.plot_name }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <StatusBadge :value="detailOrder.status" :meta-map="MaintenanceStatusMeta" />
          </el-descriptions-item>
          <el-descriptions-item label="采样日期">{{ formatDate(detailOrder.sampled_at) }}</el-descriptions-item>
          <el-descriptions-item label="pH 值">{{ Number(detailOrder.ph_value).toFixed(2) }}</el-descriptions-item>
          <el-descriptions-item label="肥力问题">{{ FertilityIssueText[detailOrder.fertility_issue] || detailOrder.fertility_issue }}</el-descriptions-item>
          <el-descriptions-item label="登记人">{{ detailOrder.operator_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="处理建议" :span="2">{{ detailOrder.suggestion }}</el-descriptions-item>
          <el-descriptions-item v-if="detailOrder.status === 'completed'" label="实际措施" :span="2">{{ detailOrder.actual_measures }}</el-descriptions-item>
          <el-descriptions-item v-if="detailOrder.status === 'completed'" label="完成日期">{{ formatDate(detailOrder.completed_at) }}</el-descriptions-item>
          <el-descriptions-item v-if="detailOrder.status === 'cancelled'" label="取消原因" :span="2">{{ detailOrder.cancel_reason }}</el-descriptions-item>
          <el-descriptions-item v-if="detailOrder.status === 'cancelled'" label="取消时间">{{ formatDate(detailOrder.cancelled_at) }}</el-descriptions-item>
        </el-descriptions>

        <!-- 认养人进度条 -->
        <div v-if="detailOrder.status === 'pending' || detailOrder.status === 'processing'" style="margin: 16px 0">
          <el-steps :active="MaintenanceProgressStage[detailOrder.status]" align-center>
            <el-step title="采样登记" :description="formatDate(detailOrder.sampled_at)" />
            <el-step title="养护处理中" description="管理员处理土壤问题" />
            <el-step title="恢复可种植" description="完成后自动恢复" />
          </el-steps>
        </div>

        <el-divider content-position="left">本地块养护历史（共 {{ historyOrders.length }} 张）</el-divider>
        <el-timeline>
          <el-timeline-item
            v-for="h in historyOrders"
            :key="h.id"
            :type="timelineType(h.status)"
            :timestamp="`采样 ${formatDate(h.sampled_at)}｜登记于 ${h.created_at}`"
          >
            <el-card shadow="never" size="small">
              <div style="display:flex; justify-content:space-between; align-items:center">
                <strong>#{{ h.id }} {{ FertilityIssueText[h.fertility_issue] || h.fertility_issue }}（pH {{ Number(h.ph_value).toFixed(1) }}）</strong>
                <StatusBadge :value="h.status" :meta-map="MaintenanceStatusMeta" />
              </div>
              <div class="muted" style="margin: 6px 0">建议：{{ h.suggestion }}</div>
              <div v-if="h.status === 'completed'">实际措施：{{ h.actual_measures }}<br />完成日期：{{ formatDate(h.completed_at) }}</div>
              <div v-else-if="h.status === 'cancelled'" class="cancel-text">已取消：{{ h.cancel_reason }}（{{ formatDate(h.cancelled_at) }}）</div>
              <div v-else class="muted">养护处理期间暂不能创建新的种植计划</div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
        <el-empty v-if="!historyOrders.length" description="暂无养护历史" :image-size="60" />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useMaintenanceStore } from '@/stores/maintenance'
import { listMaintenanceHistoryByPlot, type MaintenanceOrder } from '@/api/maintenance'
import { listPlots, type Plot } from '@/api/plot'
import { useAuth } from '@/hooks/useAuth'
import { usePagination } from '@/hooks/usePagination'
import DataTable from '@/components/DataTable.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { FertilityIssueText, MaintenanceStatusMeta, MaintenanceProgressStage } from '@/constants'
import { formatDate } from '@/utils/format'

const store = useMaintenanceStore()
const pagination = usePagination()
const route = useRoute()
const { user, isAdmin } = useAuth()

const allPlots = ref<Plot[]>([])
const queryPlotId = Number(route.query.plot_id)
const filterPlotId = ref<number | undefined>(Number.isFinite(queryPlotId) && queryPlotId > 0 ? queryPlotId : undefined)
const filterStatus = ref<string>('')

const adoptedPlots = computed(() => allPlots.value.filter((p) => p.adopter_id !== null && p.status !== 'maintaining'))
const selectablePlots = computed(() => (isAdmin.value ? allPlots.value : allPlots.value.filter((p) => p.adopter_id === user.value?.id)))

const createVisible = ref(false)
const creating = ref(false)
const today = new Date().toISOString().slice(0, 10)
const createForm = reactive({
  plot_id: undefined as number | undefined,
  sampled_at: today,
  ph_value: 6.5,
  fertility_issue: 'acidic',
  suggestion: ''
})

const completeVisible = ref(false)
const completeTarget = ref<MaintenanceOrder | null>(null)
const completeForm = reactive({ actual_measures: '', completed_at: today })

const cancelVisible = ref(false)
const cancelTarget = ref<MaintenanceOrder | null>(null)
const cancelReason = ref('')
const acting = ref(false)

const detailVisible = ref(false)
const detailOrder = ref<MaintenanceOrder | null>(null)
const historyOrders = ref<MaintenanceOrder[]>([])

function disableFuture(d: Date) {
  return d.getTime() > Date.now()
}

function timelineType(status: string): 'primary' | 'success' | 'info' | 'warning' {
  if (status === 'completed') return 'success'
  if (status === 'cancelled') return 'info'
  if (status === 'processing') return 'primary'
  return 'warning'
}

async function loadPlots() {
  const data = await listPlots({ page: 1, page_size: 200 })
  allPlots.value = data.list
}

async function fetchOrders() {
  await store.fetchOrders({
    page: pagination.page.value,
    page_size: pagination.size.value,
    plot_id: filterPlotId.value,
    status: filterStatus.value || undefined
  })
}

function reload() {
  pagination.page.value = 1
  fetchOrders()
}

function resetFilter() {
  filterPlotId.value = undefined
  filterStatus.value = ''
  reload()
}

function onPage(page: number) {
  pagination.page.value = page
  fetchOrders()
}

async function openCreate() {
  await loadPlots()
  Object.assign(createForm, { plot_id: undefined, sampled_at: today, ph_value: 6.5, fertility_issue: 'acidic', suggestion: '' })
  createVisible.value = true
}

async function submitCreate() {
  if (!createForm.plot_id) {
    ElMessage.warning('请选择已认养地块')
    return
  }
  if (!createForm.suggestion.trim()) {
    ElMessage.warning('请填写处理建议')
    return
  }
  creating.value = true
  try {
    await store.create({
      plot_id: createForm.plot_id,
      sampled_at: createForm.sampled_at,
      ph_value: createForm.ph_value,
      fertility_issue: createForm.fertility_issue,
      suggestion: createForm.suggestion.trim()
    })
    ElMessage.success('养护单已登记，地块进入养护处理期')
    createVisible.value = false
    await fetchOrders()
  } finally {
    creating.value = false
  }
}

async function onStart(row: MaintenanceOrder) {
  await store.start(row.id)
  ElMessage.success('养护单已开始处理')
  await fetchOrders()
}

function openComplete(row: MaintenanceOrder) {
  completeTarget.value = row
  completeForm.actual_measures = ''
  completeForm.completed_at = today
  completeVisible.value = true
}

async function submitComplete() {
  if (!completeTarget.value) return
  if (!completeForm.actual_measures.trim()) {
    ElMessage.warning('请填写实际措施')
    return
  }
  acting.value = true
  try {
    await store.complete(completeTarget.value.id, {
      actual_measures: completeForm.actual_measures.trim(),
      completed_at: completeForm.completed_at
    })
    ElMessage.success('养护已完成，地块恢复可种植')
    completeVisible.value = false
    await fetchOrders()
  } finally {
    acting.value = false
  }
}

function openCancel(row: MaintenanceOrder) {
  cancelTarget.value = row
  cancelReason.value = ''
  cancelVisible.value = true
}

async function submitCancel() {
  if (!cancelTarget.value) return
  if (!cancelReason.value.trim()) {
    ElMessage.warning('取消必须填写原因')
    return
  }
  acting.value = true
  try {
    await store.cancel(cancelTarget.value.id, cancelReason.value.trim())
    ElMessage.success('养护单已取消，地块恢复可种植')
    cancelVisible.value = false
    await fetchOrders()
  } finally {
    acting.value = false
  }
}

async function openDetail(row: MaintenanceOrder) {
  detailOrder.value = row
  detailVisible.value = true
  historyOrders.value = []
  try {
    historyOrders.value = await listMaintenanceHistoryByPlot(row.plot_id)
  } catch {
    historyOrders.value = []
  }
}

onMounted(async () => {
  try {
    await loadPlots()
  } catch {
    allPlots.value = []
  }
  await fetchOrders()
})
</script>

<style scoped>
.cancel-text { color: #909399; }
</style>
