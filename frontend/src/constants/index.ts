// 与后端 internal/constants/enums.go 对应的共享枚举（新增枚举值需前后端同步修改 ≥10 处）

export type RoleType = 'admin' | 'farmer' | 'citizen'
export const RoleText: Record<string, string> = {
  admin: '管理员',
  farmer: '农场主',
  citizen: '城市居民'
}

export type PlotStatus = 'available' | 'adopted' | 'harvested' | 'caring'
export const PlotStatusMeta: Record<string, { label: string; type: 'success' | 'warning' | 'info' | 'danger' | 'primary' }> = {
  available: { label: '空闲可认养', type: 'success' },
  adopted: { label: '已认养', type: 'warning' },
  harvested: { label: '待释放', type: 'info' },
  caring: { label: '养护中', type: 'danger' }
}

export type PlanStatus = 'planned' | 'planting' | 'growing' | 'harvesting' | 'completed'
export const PlanStatusMeta: Record<string, { label: string; type: 'success' | 'warning' | 'info' | 'danger' | 'primary' }> = {
  planned: { label: '已计划', type: 'info' },
  planting: { label: '播种中', type: 'primary' },
  growing: { label: '生长中', type: 'warning' },
  harvesting: { label: '采收中', type: 'danger' },
  completed: { label: '已完成', type: 'success' }
}
// 状态机（与后端 PlanStatusTransitions 对应，驱动按钮显隐）
export const PlanStatusNext: Record<string, string> = {
  planned: 'planting',
  planting: 'growing',
  growing: 'harvesting',
  harvesting: 'completed',
  completed: ''
}
export const PlanStatusActions: Record<string, string> = {
  planned: '开始播种',
  planting: '进入生长期',
  growing: '开始采收',
  harvesting: '标记完成',
  completed: ''
}

export type CropType = 'vegetable' | 'fruit' | 'herb'
export const CropTypeText: Record<string, string> = {
  vegetable: '蔬菜',
  fruit: '水果',
  herb: '香草'
}

export type Season = 'spring' | 'summer' | 'autumn' | 'winter'
export const SeasonText: Record<string, string> = {
  spring: '春季',
  summer: '夏季',
  autumn: '秋季',
  winter: '冬季'
}

export type DiaryAction = 'sowing' | 'watering' | 'fertilizing' | 'pest_control' | 'harvest' | 'other'
export const DiaryActionText: Record<string, string> = {
  sowing: '播种',
  watering: '浇水',
  fertilizing: '施肥',
  pest_control: '除虫',
  harvest: '收成',
  other: '其他'
}

export type PostType = 'experience' | 'pest' | 'recipe' | 'activity'
export const PostTypeText: Record<string, string> = {
  experience: '种植经验',
  pest: '病虫害防治',
  recipe: '食谱创意',
  activity: '线下农耕活动'
}

export type HarvestQuality = 'excellent' | 'good' | 'fair'
export const HarvestQualityText: Record<string, string> = {
  excellent: '优',
  good: '良',
  fair: '一般'
}

// 土壤养护单状态机（与后端 CareStatusTransitions 对应，驱动按钮显隐）
export type CareStatus = 'pending' | 'in_progress' | 'completed' | 'cancelled'
export const CareStatusMeta: Record<string, { label: string; type: 'success' | 'warning' | 'info' | 'danger' | 'primary' }> = {
  pending: { label: '待处理', type: 'warning' },
  in_progress: { label: '处理中', type: 'primary' },
  completed: { label: '已完成', type: 'success' },
  cancelled: { label: '已取消', type: 'info' }
}
// 当前状态可执行的动作（按钮显隐）
export const CareStatusActions: Record<string, Array<{ action: string; label: string; type: 'primary' | 'success' | 'danger' }>> = {
  pending: [
    { action: 'start', label: '开始处理', type: 'primary' },
    { action: 'cancel', label: '取消养护单', type: 'danger' }
  ],
  in_progress: [
    { action: 'complete', label: '完成养护', type: 'success' },
    { action: 'cancel', label: '取消养护单', type: 'danger' }
  ],
  completed: [],
  cancelled: []
}

export const SoilTypeText: Record<string, string> = {
  loam: '壤土',
  clay: '黏土',
  sand: '沙土',
  black: '黑土'
}

export const SunlightText: Record<string, string> = {
  full: '全日照',
  partial: '半日照',
  shade: '遮阴'
}
