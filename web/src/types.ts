export type GapItem = {
  field: string
  current_value: string
  required_value: string
  description?: string
  description_fa?: string
  advice_fa?: string
}

export type ProductMatch = {
  product_id: string
  product_name: string
  product_name_fa: string
  eligible: boolean
  is_conditional?: boolean
  conditions_fa?: string[]
  reasons?: string[]
  reasons_fa?: string[]
  gaps?: GapItem[]
  advice_fa?: string[]
  alternatives_fa?: string[]
  obligations_fa?: string[]
  credit_limit_fa?: string
  circular_refs?: string[]
  score: number
}

export type DefaultWarning = {
  current_risk_level: string
  potential_risk_level: string
  consequences?: string[]
  consequences_fa?: string[]
}

export type MatchResponse = {
  customer_id: string
  national_id: string
  customer_name: string
  is_existing: boolean
  is_cold_start: boolean
  risk_level: string
  risk_score: number
  risk_reason?: string
  visit_purpose?: string
  eligible_products: ProductMatch[]
  ineligible_products: ProductMatch[]
  personalized_offers: ProductMatch[]
  notes_fa?: string[]
  default_warning?: DefaultWarning | null
  upstream_errors?: string[]
}

export type IdentityProfile = {
  customer_id: string
  national_id: string
  name: string
  age: number
  gender: string
  occupation: string
  employment_type: string
  customer_type: string
  account_open_date: string
  is_existing: boolean
}

export type FinancialProfile = {
  customer_id: string
  monthly_income: number
  account_turnover_3m: number
  account_turnover_12m: number
  total_deposits: number
  active_loans: number
  total_loan_amount: number
  installment_default: number
  spending_pattern: string
  payment_history: string
  has_guarantor: boolean
}

export type RiskAssessment = {
  customer_id: string
  risk_level: string
  risk_score: number
  reason: string
  is_cold_start: boolean
}

export type CustomerRecord = {
  identity: IdentityProfile
  financial: FinancialProfile
  risk: RiskAssessment
}

export type ApiError = {
  error?: string
  error_fa?: string
  upstream_errors?: string[]
}

export type ChatMessage = {
  id: string
  role: 'user' | 'assistant'
  content: string
}

export type SampleCustomer = {
  nationalId: string
  name: string
  note: string
}
