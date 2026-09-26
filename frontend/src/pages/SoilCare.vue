<template>
  <div class="page-card">
    <div style="display: flex; justify-content: space-between; align-items: center">
      <h3 class="page-title">土壤养护单</h3>
      <el-button v-if="isAdmin" type="primary" @click="openCreate">+ 登记养护单</el-button>
    </div>

    <el-card shadow="never" style="margin-bottom: 16px">
      <el-form inline>
        <el-form-item label="地块">
          <el-select v-model="plotFilter" placeholder="全部地块" clearable style="width: 220px" @change="fetch">
            <el-option v-for="p in plotOptions" :key="p.id" :label="`${p.code} ${p.name}`" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="statusFilter" placeholder="全部状态" clearable style="width: 160px" @change="fetch">
            <el-option v-for="(m, k) in CareStatusMeta" :key="k" :label="m.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-empty v-if="!store.loading && !groups.length" description="暂无养护记录" />

    <el-collapse v-else v-model="activePlots" v-loading="store.loading">
      <el-collapse-item v-for="g in groups" :key="g.plotId" :name="g.plotId">
        <template #title>
          <div class="plot-group-title">
            <span class="plot-group-name">{{ g.plotCode }} {{ g.plotName }}</span>
            <el-tag size="small" type="info" effect="plain">共 {{ g.orders.length }} 条</el-tag>
            <el-tag v-if="g.open" size="small" type="danger">养护中 · 暂停种植</el-tag>
          </div>
        </template>

        <el-timeline>
          <el-timeline-item
            v-for="o in g.orders"
            :key="o.id"
            :type="dotType(o.status)"
            :timestamp="`登记于 ${o.created_at}`"
            placement="top"
          >
            <el-card shadow="hover" class="care-card">
              <div class="care-card-head">
                <StatusBadge :value="o.status" :meta-map="CareStatusMeta" />
                <span class="muted">养护单 #{{ o.id }} · 经办人 {{ o.admin_name || '-' }}</span>
              </div>
              <el-descriptions :column="2" border size="small" style="margin-top: 8px">
                <el-descriptions-item label="采样日期">{{ formatDate(o.sampled_date) }}</el-descriptions-item>
                <el-descriptions-item label="pH 值">{{ o.ph_value }}</el-descriptions-item>
                <el-descriptions-item label="肥力问题" :span="2">{{ o.fertility_issue }}</el-descriptions-item>
                <el-descriptions-item label="处理建议" :span="2">{{ o.treatment_advice }}</el-descriptions-item>
                <el-descriptions-item v-if="o.actual_measures" label="实际措施" :span="2">{{ o.actual_measures }}</el-descriptions-item>
                <el-descriptions-item v-if="o.completed_date" label="完成日期">{{ formatDate(o.completed_date) }}</el-descriptions-item>
                <el-descriptions-item v-if="o.cancel_reason" label="取消原因" :span="2">
                  <span style="color: #f56c6c">{{ o.cancel_reason }}</span>
                </el-descriptions-item>
              </el-descriptions>

              <div v-if="isAdmin && CareStatusActions[o.status]?.length" class="care-actions">
                <el-button
                  v-for="act in CareStatusActions[o.status]"
                  :key="act.action"
                  :type="act.type"
                  size="small"
                  @click="onAction(act.action, o)"
                >{{ act.label }}</el-button>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </el-collapse-item>
    </el-collapse>

    <!-- 登记养护单 -->
    <el-dialog v-model="createVisible" title="登记土壤养护单" width="560px">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="100px">
        <el-form-item label="已认养地块" prop="plot_id">
          <el-select v-model="createForm.plot_id" placeholder="选择已认养地块" style="width: 100%">
            <el-option v-for="p in adoptedPlots" :key="p.id" :label="`${p.code} ${p.name}`" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="采样日期" prop="sampled_date">
          <el-date-picker v-model="createForm.sampled_date" type="date" value-format="YYYY-MM-DD" placeholder="选择采样日期" style="width: 100%" />
        </el-form-item>
        <el-form-item label="pH 值" prop="ph_value">
          <el-input-number v-model="createForm.ph_value" :min="0" :max="14" :step="0.1" :precision="2" />
        </el-form-item>
        <el-form-item label="肥力问题" prop="fertility_issue">
          <el-input v-model="createForm.fertility_issue" type="textarea" placeholder="如 土壤板结、有机质偏低、缺磷偏酸" />
        </el-form-item>
        <el-form-item label="处理建议" prop="treatment_advice">
          <el-input v-model="createForm.treatment_advice" type="textarea" placeholder="如 深翻、增施腐熟堆肥、撒生石灰调酸" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">保存并进入养护期</el-button>
      </template>
    </el-dialog>

    <!-- 完成养护 -->
    <el-dialog v-model="completeVisible" title="完成土壤养护" width="520px">
      <el-form label-width="100px">
        <el-form-item label="实际措施" required>
          <el-input v-model="completeForm.actual_measures" type="textarea" placeholder="填写实际采取的养护措施" />
        </el-form-item>
        <el-form-item label="完成日期">
          <el-date-picker v-model="completeForm.completed_date" type="date" value-format="YYYY-MM-DD" placeholder="默认今天" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="completeVisible = false">取消</el-button>
        <el-button type="success" :loading="submitting" @click="submitComplete">确认完成（地块恢复可种植）</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { useCareStore } from '@/stores/soilCare'
import { listAdoptedPlotsForCare, type SoilCareOrder } from '@/api/soilCare'
import { listPlots } from '@/api/plot'
import { useAuth } from '@/hooks/useAuth'
import StatusBadge from '@/components/StatusBadge.vue'
import { CareStatusMeta, CareStatusActions } from '@/constants'
import { formatDate } from '@/utils/format'

const store = useCareStore()
const { isAdmin } = useAuth()
const route = useRoute()

const plotFilter = ref<number | undefined>(route.query.plot_id ? Number(route.query.plot_id) : undefined)
const statusFilter = ref<string>('')
const activePlots = ref<number[]>([])
const plotOptions = ref<Array<{ id: number; code: string; name: string }>>([])
const adoptedPlots = ref<Array<{ id: number; code: string; name: string }>>([])

const createVisible = ref(false)
const completeVisible = ref(false)
const submitting = ref(false)
const createFormRef = ref<FormInstance>()
const activeOrder = ref<SoilCareOrder | null>(null)

const today = new Date().toISOString().slice(0, 10)
const createForm = reactive({
  plot_id: undefined as number | undefined,
  sampled_date: today,
  ph_value: 6.5,
  fertility_issue: '',
  treatment_advice: ''
})
const createRules = {
  plot_id: [{ required: true, message: '请选择已认养地块', trigger: 'change' }],
  sampled_date: [{ required: true, message: '请选择采样日期', trigger: 'change' }],
  ph_value: [{ required: true, message: '请填写 pH 值', trigger: 'blur' }],
  fertility_issue: [{ required: true, message: '请描述肥力问题', trigger: 'blur' }],
  treatment_advice: [{ required: true, message: '请填写处理建议', trigger: 'blur' }]
}
const completeForm = reactive({ actual_measures: '', completed_date: today })

// 按地块分组展示历史记录
const groups = computed(() => {
  const map = new Map<number, { plotId: number; plotCode: string; plotName: string; orders: SoilCareOrder[]; open: boolean }>()
  for (const o of store.orders) {
    let g = map.get(o.plot_id)
    if (!g) {
      g = { plotId: o.plot_id, plotCode: o.plot_code, plotName: o.plot_name, orders: [], open: false }
      map.set(o.plot_id, g)
    }
    g.orders.push(o)
    if (o.status === 'pending' || o.status === 'in_progress') g.open = true
  }
  const list = Array.from(map.values())
  list.sort((a, b) => Number(b.open) - Number(a.open) || a.plotCode.localeCompare(b.plotCode))
  return list
})

function dotType(status: string): 'primary' | 'success' | 'info' | 'warning' | 'danger' {
  if (status === 'completed') return 'success'
  if (status === 'cancelled') return 'info'
  if (status === 'in_progress') return 'primary'
  return 'warning'
}

async function fetch() {
  await store.fetchOrders({
    page: 1,
    page_size: 200,
    plot_id: plotFilter.value || undefined,
    status: statusFilter.value || undefined
  })
  activePlots.value = groups.value.map((g) => g.plotId)
}

function resetFilter() {
  plotFilter.value = undefined
  statusFilter.value = ''
  fetch()
}

async function loadPlotOptions() {
  try {
    const data = await listPlots({ page: 1, page_size: 200 })
    plotOptions.value = data.list.map((p) => ({ id: p.id, code: p.code, name: p.name }))
  } catch {
    plotOptions.value = []
  }
}

async function openCreate() {
  createForm.plot_id = undefined
  createForm.sampled_date = today
  createForm.ph_value = 6.5
  createForm.fertility_issue = ''
  createForm.treatment_advice = ''
  createVisible.value = true
  try {
    adoptedPlots.value = await listAdoptedPlotsForCare()
  } catch {
    adoptedPlots.value = []
  }
}

async function submitCreate() {
  if (!createFormRef.value) return
  await createFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await store.create({ ...createForm, plot_id: createForm.plot_id as number })
      ElMessage.success('养护单已登记，地块进入养护处理期')
      createVisible.value = false
      await fetch()
      await loadPlotOptions()
    } finally {
      submitting.value = false
    }
  })
}

async function onAction(action: string, order: SoilCareOrder) {
  if (action === 'start') {
    await store.start(order.id)
    ElMessage.success('养护单已开始处理')
    await fetch()
  } else if (action === 'complete') {
    activeOrder.value = order
    completeForm.actual_measures = ''
    completeForm.completed_date = today
    completeVisible.value = true
  } else if (action === 'cancel') {
    try {
      const { value } = await ElMessageBox.prompt('请填写取消原因（必填）', '取消养护单', {
        confirmButtonText: '确认取消',
        cancelButtonText: '返回',
        inputType: 'textarea',
        inputValidator: (v) => (v && v.trim() ? true : '取消原因不能为空')
      })
      await store.cancel(order.id, value.trim())
      ElMessage.success('养护单已取消，地块恢复可种植')
      await fetch()
    } catch {
      /* 用户放弃 */
    }
  }
}

async function submitComplete() {
  if (!activeOrder.value) return
  if (!completeForm.actual_measures.trim()) {
    ElMessage.warning('请填写实际措施')
    return
  }
  submitting.value = true
  try {
    await store.complete(activeOrder.value.id, {
      actual_measures: completeForm.actual_measures.trim(),
      completed_date: completeForm.completed_date || today
    })
    ElMessage.success('养护已完成，地块恢复可种植')
    completeVisible.value = false
    await fetch()
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetch()
  loadPlotOptions()
})
</script>

<style scoped>
.plot-group-title { display: flex; align-items: center; gap: 10px; }
.plot-group-name { font-weight: 600; color: #303133; }
.care-card-head { display: flex; align-items: center; justify-content: space-between; }
.care-actions { margin-top: 10px; text-align: right; }
.muted { color: #909399; font-size: 12px; }
</style>
