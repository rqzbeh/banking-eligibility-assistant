import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { agentChat, deleteCustomer, healthCheck, listCustomers, matchColdStart, matchCustomer, saveCustomer } from './api'
import { MatchResults } from './components/MatchResults'
import type { ChatMessage, CustomerRecord, MatchResponse, SampleCustomer } from './types'

type Mode = 'quick' | 'clients' | 'agent'

const SAMPLES: SampleCustomer[] = [
  { nationalId: '0012345678', name: 'فاطمه احمدی', note: 'خانه‌دار — درآمد پایین' },
  { nationalId: '0023456789', name: 'علی رضایی', note: 'مدیر — واجد شرایط' },
  { nationalId: '0034567890', name: 'محمد حسینی', note: 'کارمند — اقساط معوق' },
  { nationalId: '0045678901', name: 'زهرا کریمی', note: 'کارمند خصوصی' },
  { nationalId: '0056789012', name: 'رضا محمدی', note: 'بازنشسته' },
]

const OCC_LABEL: Record<string, string> = {
  employee: 'کارمند',
  self_employed: 'شغل آزاد',
  housewife: 'خانه‌دار',
  manager: 'مدیر',
  retired: 'بازنشسته',
  unemployed: 'بیکار',
  student: 'دانشجو',
}

const EMP_LABEL: Record<string, string> = {
  government: 'دولتی',
  private: 'خصوصی',
  freelance: 'آزاد',
  none: 'بدون اشتغال',
}

function uid() {
  return Math.random().toString(36).slice(2, 10)
}

function emptyCustomer(): CustomerRecord {
  return {
    identity: {
      customer_id: '',
      national_id: '',
      name: '',
      age: 30,
      gender: 'male',
      occupation: 'employee',
      employment_type: 'private',
      customer_type: 'real',
      account_open_date: '',
      is_existing: true,
    },
    financial: {
      customer_id: '',
      monthly_income: 0,
      account_turnover_3m: 0,
      account_turnover_12m: 0,
      total_deposits: 0,
      active_loans: 0,
      total_loan_amount: 0,
      installment_default: 0,
      spending_pattern: 'moderate',
      payment_history: 'good',
      has_guarantor: false,
    },
    risk: {
      customer_id: '',
      risk_level: 'medium',
      risk_score: 50,
      reason: '',
      is_cold_start: false,
    },
  }
}

export default function App() {
  const [mode, setMode] = useState<Mode>('quick')
  const [backendOk, setBackendOk] = useState(false)
  const [nationalId, setNationalId] = useState('')
  const [visitPurpose, setVisitPurpose] = useState('')
  const [includeDefault, setIncludeDefault] = useState(true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<MatchResponse | null>(null)
  const [needColdStart, setNeedColdStart] = useState(false)

  // cold-start form
  const [csName, setCsName] = useState('')
  const [csAge, setCsAge] = useState(30)
  const [csGender, setCsGender] = useState('female')
  const [csOcc, setCsOcc] = useState('employee')
  const [csEmp, setCsEmp] = useState('private')
  const [csIncome, setCsIncome] = useState(20_000_000)
  const [csPurpose, setCsPurpose] = useState('وام شخصی')

  // chat
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [threadId, setThreadId] = useState<string | undefined>()
  const [chatInput, setChatInput] = useState('')
  const [chatLoading, setChatLoading] = useState(false)
  const [customers, setCustomers] = useState<CustomerRecord[]>([])
  const [customerForm, setCustomerForm] = useState<CustomerRecord>(() => emptyCustomer())
  const [editingNationalId, setEditingNationalId] = useState<string | undefined>()
  const [customerLoading, setCustomerLoading] = useState(false)

  useEffect(() => {
    let alive = true
    const tick = async () => {
      const ok = await healthCheck()
      if (alive) setBackendOk(ok)
    }
    tick()
    const id = window.setInterval(tick, 15000)
    return () => {
      alive = false
      window.clearInterval(id)
    }
  }, [])

  useEffect(() => {
    if (mode === 'clients') void refreshCustomers()
  }, [mode])

  const backendLabel = useMemo(
    () => (backendOk ? 'بک‌اند متصل' : 'بک‌اند قطع'),
    [backendOk],
  )

  async function runMatch(e?: FormEvent) {
    e?.preventDefault()
    const nid = nationalId.trim()
    if (!nid) {
      setError('کد ملی را وارد کنید.')
      return
    }
    if (!backendOk) {
      setError('بک‌اند در دسترس نیست.')
      return
    }
    setLoading(true)
    setError(null)
    setNeedColdStart(false)
    try {
      const res = await matchCustomer({
        national_id: nid,
        visit_purpose: visitPurpose.trim() || undefined,
        include_default_warning: includeDefault,
      })
      if (res.ok) {
        setResult(res.data)
        setNeedColdStart(false)
      } else if (res.status === 404) {
        setResult(null)
        setNeedColdStart(true)
        setCsPurpose(visitPurpose.trim() || 'وام شخصی')
      } else {
        setResult(null)
        setError(res.message)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطای شبکه')
    } finally {
      setLoading(false)
    }
  }

  async function runColdStart(e: FormEvent) {
    e.preventDefault()
    if (!csName.trim()) {
      setError('نام الزامی است.')
      return
    }
    setLoading(true)
    setError(null)
    try {
      const res = await matchColdStart({
        name: csName.trim(),
        age: csAge,
        gender: csGender,
        occupation: csOcc,
        employment_type: csEmp,
        approx_income: csIncome,
        visit_purpose: csPurpose,
        include_default_warning: includeDefault,
      })
      if (res.ok) {
        setResult(res.data)
        setNeedColdStart(false)
      } else {
        setError(res.message)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطای شبکه')
    } finally {
      setLoading(false)
    }
  }

  async function sendChat(e?: FormEvent) {
    e?.preventDefault()
    const text = chatInput.trim()
    if (!text || chatLoading) return
    const userMsg: ChatMessage = { id: uid(), role: 'user', content: text }
    setMessages((m) => [...m, userMsg])
    setChatInput('')
    setChatLoading(true)
    try {
      const res = await agentChat({ message: text, thread_id: threadId })
      if (res.ok) {
        setThreadId(res.thread_id)
        setMessages((m) => [...m, { id: uid(), role: 'assistant', content: res.reply }])
      } else {
        setMessages((m) => [
          ...m,
          { id: uid(), role: 'assistant', content: `خطا: ${res.message}` },
        ])
      }
    } finally {
      setChatLoading(false)
    }
  }

  function pickSample(s: SampleCustomer) {
    setNationalId(s.nationalId)
    setVisitPurpose('')
    setResult(null)
    setNeedColdStart(false)
    setError(null)
    setMode('quick')
  }

  async function refreshCustomers() {
    setCustomerLoading(true)
    setError(null)
    try {
      const res = await listCustomers()
      if (res.ok) setCustomers(res.data)
      else setError(res.message)
    } finally {
      setCustomerLoading(false)
    }
  }

  function editCustomer(c: CustomerRecord) {
    setCustomerForm(JSON.parse(JSON.stringify(c)) as CustomerRecord)
    setEditingNationalId(c.identity.national_id)
  }

  function updateCustomerForm(path: string, value: string | number | boolean) {
    setCustomerForm((cur) => {
      const next = JSON.parse(JSON.stringify(cur)) as CustomerRecord
      const [section, field] = path.split('.') as [keyof CustomerRecord, string]
      ;(next[section] as Record<string, unknown>)[field] = value
      if (field === 'customer_id') {
        next.financial.customer_id = String(value)
        next.risk.customer_id = String(value)
      }
      return next
    })
  }

  async function submitCustomer(e: FormEvent) {
    e.preventDefault()
    const record = {
      ...customerForm,
      financial: { ...customerForm.financial, customer_id: customerForm.identity.customer_id },
      risk: { ...customerForm.risk, customer_id: customerForm.identity.customer_id },
    }
    setCustomerLoading(true)
    setError(null)
    try {
      const res = await saveCustomer(record, editingNationalId)
      if (res.ok) {
        setCustomerForm(emptyCustomer())
        setEditingNationalId(undefined)
        await refreshCustomers()
      } else {
        setError(res.message)
      }
    } finally {
      setCustomerLoading(false)
    }
  }

  async function removeCustomer(nationalId: string) {
    setCustomerLoading(true)
    setError(null)
    try {
      const res = await deleteCustomer(nationalId)
      if (res.ok) {
        if (editingNationalId === nationalId) {
          setCustomerForm(emptyCustomer())
          setEditingNationalId(undefined)
        }
        await refreshCustomers()
      } else {
        setError(res.message)
      }
    } finally {
      setCustomerLoading(false)
    }
  }

  return (
    <div className="app-shell">
      <header className="hero">
        <div className="hero-kicker">سامانه شعب · تعیین اهلیت و پیشنهاد محصول</div>
        <h1>دستیار هوشمند بانکی</h1>
        <p>
          بررسی سریع اهلیت بر اساس بخشنامه‌ها، تحلیل شکاف، افر شخصی‌سازی‌شده و مسیر جایگزین —
          با استناد و خروجی ساخت‌یافته برای کارمند شعبه.
        </p>
      </header>

      <div className="layout">
        <aside className="sidebar">
          <div className={`status ${backendOk ? 'up' : 'down'}`}>
            <span className="dot" aria-hidden />
            {backendLabel}
          </div>

          <div className="side-block">
            <h3>حالت کار</h3>
            <div className="mode-switch" role="tablist" aria-label="حالت کار">
              <button
                type="button"
                className={mode === 'quick' ? 'active' : ''}
                onClick={() => setMode('quick')}
              >
                بررسی سریع
              </button>
              <button
                type="button"
                className={mode === 'agent' ? 'active' : ''}
                onClick={() => setMode('agent')}
              >
                دستیار هوشمند
              </button>
            </div>
            <button
              type="button"
              className="btn btn-secondary"
              style={{ width: '100%', marginTop: '0.55rem' }}
              onClick={() => setMode('clients')}
            >
              مدیریت مشتریان
            </button>
          </div>

          <div className="side-block">
            <h3>مشتریان نمونه</h3>
            <div className="sample-list">
              {SAMPLES.map((s) => (
                <button
                  key={s.nationalId}
                  type="button"
                  className="sample-btn"
                  onClick={() => pickSample(s)}
                >
                  <strong>{s.name}</strong>
                  <span>
                    {s.nationalId} · {s.note}
                  </span>
                </button>
              ))}
            </div>
            <div className="chips">
              <span className="chip">کد ملی ۱۰ رقمی</span>
              <span className="chip">تحلیل شکاف</span>
              <span className="chip">افر مشروط</span>
              <span className="chip">بخشنامه‌محور</span>
            </div>
          </div>
        </aside>

        <main>
          {mode === 'quick' ? (
            <>
              <section className="panel">
                <p className="panel-title">ورودی بررسی</p>
                <form onSubmit={runMatch}>
                  <div className="form-grid">
                    <div className="field">
                      <label htmlFor="nid">کد ملی مشتری</label>
                      <input
                        id="nid"
                        inputMode="numeric"
                        autoComplete="off"
                        placeholder="مثال: 0012345678"
                        value={nationalId}
                        onChange={(e) => setNationalId(e.target.value)}
                      />
                    </div>
                    <div className="field">
                      <label htmlFor="purpose">هدف مراجعه (اختیاری)</label>
                      <input
                        id="purpose"
                        placeholder="دسته‌چک، وام شخصی، سپرده…"
                        value={visitPurpose}
                        onChange={(e) => setVisitPurpose(e.target.value)}
                      />
                    </div>
                  </div>
                  <div className="row-actions">
                    <button
                      className="btn btn-primary"
                      type="submit"
                      disabled={loading || !backendOk}
                    >
                      {loading ? 'در حال بررسی…' : 'بررسی اهلیت'}
                    </button>
                    <button
                      className="btn btn-secondary"
                      type="button"
                      onClick={() => {
                        setResult(null)
                        setNeedColdStart(false)
                        setError(null)
                      }}
                    >
                      پاک‌سازی نتیجه
                    </button>
                    <label className="check">
                      <input
                        type="checkbox"
                        checked={includeDefault}
                        onChange={(e) => setIncludeDefault(e.target.checked)}
                      />
                      هشدار عدم پرداخت
                    </label>
                  </div>
                </form>
              </section>

              {error && <div className="alert error">{error}</div>}

              {needColdStart && (
                <section className="panel" style={{ marginTop: '0.9rem' }}>
                  <div className="card warn" style={{ marginBottom: '0.85rem' }}>
                    <h4>مشتری یافت نشد</h4>
                    <p>
                      این کد ملی در سامانه‌های هویتی موجود نیست. مسیر <b>غیرمشتری</b> را با
                      اطلاعات خوداظهاری ادامه دهید. افرها مشروط به افتتاح حساب و مدارک خواهند
                      بود.
                    </p>
                  </div>
                  <form onSubmit={runColdStart}>
                    <div className="form-grid">
                      <div className="field">
                        <label htmlFor="cs-name">نام و نام خانوادگی</label>
                        <input
                          id="cs-name"
                          value={csName}
                          onChange={(e) => setCsName(e.target.value)}
                          placeholder="سارا نوروزی"
                        />
                      </div>
                      <div className="field">
                        <label htmlFor="cs-purpose">هدف مراجعه</label>
                        <input
                          id="cs-purpose"
                          value={csPurpose}
                          onChange={(e) => setCsPurpose(e.target.value)}
                        />
                      </div>
                      <div className="field">
                        <label htmlFor="cs-age">سن</label>
                        <input
                          id="cs-age"
                          type="number"
                          min={15}
                          max={100}
                          value={csAge}
                          onChange={(e) => setCsAge(Number(e.target.value))}
                        />
                      </div>
                      <div className="field">
                        <label htmlFor="cs-gender">جنسیت</label>
                        <select
                          id="cs-gender"
                          value={csGender}
                          onChange={(e) => setCsGender(e.target.value)}
                        >
                          <option value="female">زن</option>
                          <option value="male">مرد</option>
                        </select>
                      </div>
                      <div className="field">
                        <label htmlFor="cs-occ">شغل</label>
                        <select
                          id="cs-occ"
                          value={csOcc}
                          onChange={(e) => setCsOcc(e.target.value)}
                        >
                          {Object.entries(OCC_LABEL).map(([k, v]) => (
                            <option key={k} value={k}>
                              {v}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="field">
                        <label htmlFor="cs-emp">نوع اشتغال</label>
                        <select
                          id="cs-emp"
                          value={csEmp}
                          onChange={(e) => setCsEmp(e.target.value)}
                        >
                          {Object.entries(EMP_LABEL).map(([k, v]) => (
                            <option key={k} value={k}>
                              {v}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="field">
                        <label htmlFor="cs-income">درآمد ماهانه (تومان)</label>
                        <input
                          id="cs-income"
                          type="number"
                          min={0}
                          step={1_000_000}
                          value={csIncome}
                          onChange={(e) => setCsIncome(Number(e.target.value))}
                        />
                      </div>
                    </div>
                    <div className="row-actions">
                      <button className="btn btn-primary" type="submit" disabled={loading}>
                        {loading ? 'در حال ارزیابی…' : 'بررسی اهلیت اولیه'}
                      </button>
                    </div>
                  </form>
                </section>
              )}

              {result && (
                <div style={{ marginTop: '1rem' }}>
                  <MatchResults data={result} />
                </div>
              )}
            </>
          ) : mode === 'clients' ? (
            <section className="panel">
              <p className="panel-title">مدیریت ورودی‌های RBCI محلی</p>
              {error && <div className="alert error">{error}</div>}
              <form onSubmit={submitCustomer}>
                <div className="form-grid">
                  <div className="field">
                    <label htmlFor="cust-id">شناسه مشتری</label>
                    <input
                      id="cust-id"
                      value={customerForm.identity.customer_id}
                      onChange={(e) => updateCustomerForm('identity.customer_id', e.target.value)}
                      placeholder="C006"
                    />
                  </div>
                  <div className="field">
                    <label htmlFor="cust-nid">کد ملی</label>
                    <input
                      id="cust-nid"
                      value={customerForm.identity.national_id}
                      onChange={(e) => updateCustomerForm('identity.national_id', e.target.value)}
                      placeholder="1234567890"
                      disabled={!!editingNationalId}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor="cust-name">نام</label>
                    <input
                      id="cust-name"
                      value={customerForm.identity.name}
                      onChange={(e) => updateCustomerForm('identity.name', e.target.value)}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor="cust-age">سن</label>
                    <input
                      id="cust-age"
                      type="number"
                      min={15}
                      max={100}
                      value={customerForm.identity.age}
                      onChange={(e) => updateCustomerForm('identity.age', Number(e.target.value))}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor="cust-occ">شغل</label>
                    <select
                      id="cust-occ"
                      value={customerForm.identity.occupation}
                      onChange={(e) => updateCustomerForm('identity.occupation', e.target.value)}
                    >
                      {Object.entries(OCC_LABEL).map(([k, v]) => (
                        <option key={k} value={k}>
                          {v}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor="cust-income">درآمد ماهانه</label>
                    <input
                      id="cust-income"
                      type="number"
                      min={0}
                      step={1_000_000}
                      value={customerForm.financial.monthly_income}
                      onChange={(e) => updateCustomerForm('financial.monthly_income', Number(e.target.value))}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor="cust-turnover">گردش ۳ ماهه</label>
                    <input
                      id="cust-turnover"
                      type="number"
                      min={0}
                      step={1_000_000}
                      value={customerForm.financial.account_turnover_3m}
                      onChange={(e) => updateCustomerForm('financial.account_turnover_3m', Number(e.target.value))}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor="cust-defaults">اقساط معوق</label>
                    <input
                      id="cust-defaults"
                      type="number"
                      min={0}
                      value={customerForm.financial.installment_default}
                      onChange={(e) => updateCustomerForm('financial.installment_default', Number(e.target.value))}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor="cust-risk">سطح ریسک RBCI</label>
                    <select
                      id="cust-risk"
                      value={customerForm.risk.risk_level}
                      onChange={(e) => updateCustomerForm('risk.risk_level', e.target.value)}
                    >
                      <option value="low">کم</option>
                      <option value="medium">متوسط</option>
                      <option value="high">بالا</option>
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor="cust-score">امتیاز RBCI</label>
                    <input
                      id="cust-score"
                      type="number"
                      min={0}
                      max={100}
                      value={customerForm.risk.risk_score}
                      onChange={(e) => updateCustomerForm('risk.risk_score', Number(e.target.value))}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor="cust-risk-reason">دلیل RBCI</label>
                    <input
                      id="cust-risk-reason"
                      value={customerForm.risk.reason}
                      onChange={(e) => updateCustomerForm('risk.reason', e.target.value)}
                    />
                  </div>
                  <label className="check" style={{ alignSelf: 'end' }}>
                    <input
                      type="checkbox"
                      checked={customerForm.financial.has_guarantor}
                      onChange={(e) => updateCustomerForm('financial.has_guarantor', e.target.checked)}
                    />
                    ضامن دارد
                  </label>
                </div>
                <div className="row-actions">
                  <button className="btn btn-primary" type="submit" disabled={customerLoading}>
                    {editingNationalId ? 'ذخیره ویرایش' : 'افزودن مشتری'}
                  </button>
                  <button
                    className="btn btn-secondary"
                    type="button"
                    onClick={() => {
                      setCustomerForm(emptyCustomer())
                      setEditingNationalId(undefined)
                    }}
                  >
                    فرم جدید
                  </button>
                </div>
              </form>

              <div className="section-head">
                <h3>مشتریان ثبت‌شده</h3>
                <span className="count">{customers.length}</span>
              </div>
              <div className="sample-list">
                {customers.map((c) => (
                  <div className="sample-btn" key={c.identity.national_id}>
                    <strong>{c.identity.name}</strong>
                    <span>
                      {c.identity.national_id} · {c.identity.customer_id} · RBCI: {c.risk.risk_level}
                    </span>
                    <div className="row-actions" style={{ marginTop: '0.5rem' }}>
                      <button className="btn btn-secondary" type="button" onClick={() => editCustomer(c)}>
                        ویرایش
                      </button>
                      <button className="btn btn-secondary" type="button" onClick={() => void removeCustomer(c.identity.national_id)}>
                        حذف
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          ) : (
            <section className="panel">
              <div className="card info" style={{ marginBottom: '0.85rem' }}>
                <h4>حالت دستیار هوشمند</h4>
                <p>
                  مدل زبانی فقط مکالمه و توضیح می‌کند؛ منطق اهلیت همچنان قطعی و مبتنی بر موتور
                  قوانین Go است. نمونه: «کد ملی ۰۰۱۲۳۴۵۶۷۸ را برای دسته‌چک بررسی کن».
                </p>
              </div>

              <div className="chat-wrap">
                <div className="chat-log" aria-live="polite">
                  {messages.length === 0 && (
                    <div className="bubble assistant">
                      سلام. کد ملی مشتری یا مشخصات غیرمشتری را بفرستید تا اهلیت و افرها را بررسی
                      کنم.
                    </div>
                  )}
                  {messages.map((m) => (
                    <div key={m.id} className={`bubble ${m.role}`}>
                      {m.content}
                    </div>
                  ))}
                  {chatLoading && <div className="loading">در حال تحلیل…</div>}
                </div>

                <form className="chat-composer" onSubmit={sendChat}>
                  <div className="field" style={{ margin: 0 }}>
                    <label className="sr-only" htmlFor="chat">
                      پیام
                    </label>
                    <textarea
                      id="chat"
                      rows={2}
                      placeholder="پیام خود را بنویسید…"
                      value={chatInput}
                      onChange={(e) => setChatInput(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && !e.shiftKey) {
                          e.preventDefault()
                          void sendChat()
                        }
                      }}
                    />
                  </div>
                  <div style={{ display: 'grid', gap: '0.4rem' }}>
                    <button className="btn btn-primary" type="submit" disabled={chatLoading}>
                      ارسال
                    </button>
                    <button
                      className="btn btn-secondary"
                      type="button"
                      onClick={() => {
                        setMessages([])
                        setThreadId(undefined)
                      }}
                    >
                      گفتگوی جدید
                    </button>
                  </div>
                </form>
              </div>
            </section>
          )}
        </main>
      </div>
    </div>
  )
}
