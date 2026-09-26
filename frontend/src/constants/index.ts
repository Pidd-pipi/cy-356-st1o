// 与后端 internal/constants/enums.go 对应的共享枚举（新增枚举值需前后端同步修改 ≥10 处）

export type RoleType = 'admin' | 'farmer' | 'citizen'
export const RoleText: Record<string, string> = {
  admin: '管理员',
  farmer: '农场主',
  citizen: '城市居民'
}

export type PlotStatus = 'available' | 'adopted' | 'harvested' | 'maintaining'
export const PlotStatusMeta: Record<string, { label: string; type: 'success' | 'warning' | 'info' | 'danger' | 'primary' }> = {
  available: { label: '空闲可认养', type: 'success' },
  adopted: { label: '已认养', type: 'warning' },
  harvested: { label: '待释放', type: 'info' },
  maintaining: { label: '养护中', type: 'danger' }
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

// 土壤养护单状态机（与后端 MaintenanceStatusTransitions 对应，驱动按钮显隐）
export type MaintenanceStatus = 'pending' | 'processing' | 'completed' | 'cancelled'
export const MaintenanceStatusMeta: Record<string, { label: string; type: 'success' | 'warning' | 'info' | 'danger' | 'primary' }> = {
  pending: { label: '待处理', type: 'warning' },
  processing: { label: '处理中', type: 'primary' },
  completed: { label: '已完成', type: 'success' },
  cancelled: { label: '已取消', type: 'info' }
}
// 认养人视角的进度阶段（待处理 → 处理中 → 已结束）
export const MaintenanceProgressStage: Record<string, number> = {
  pending: 1,
  processing: 2,
  completed: 3,
  cancelled: 3
}

// 肥力问题类型（与后端 FertilityIssue 对应）
export type FertilityIssue =
  | 'acidic'
  | 'alkaline'
  | 'nutrient_low'
  | 'salinized'
  | 'organic_low'
  | 'drainage_poor'
  | 'healthy'
export const FertilityIssueText: Record<string, string> = {
  acidic: '土壤偏酸',
  alkaline: '土壤偏碱',
  nutrient_low: '养分不足',
  salinized: '板结盐渍化',
  organic_low: '有机质偏低',
  drainage_poor: '排水不良',
  healthy: '土壤健康'
}
