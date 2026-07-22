import type { MatchResponse } from '../types'
import { ProductCard } from './ProductCard'

function riskLabel(level?: string) {
  const lvl = (level || '').toLowerCase()
  if (lvl === 'low') return { text: 'ریسک کم', cls: 'ok' }
  if (lvl === 'medium') return { text: 'ریسک متوسط', cls: 'warn' }
  if (lvl === 'high') return { text: 'ریسک بالا', cls: 'bad' }
  return { text: level || '—', cls: 'neutral' }
}

export function MatchResults({ data }: { data: MatchResponse }) {
  const risk = riskLabel(data.risk_level)
  const offers = data.personalized_offers || []
  const elig = data.eligible_products || []
  const inelig = data.ineligible_products || []

  return (
    <section>
      <div className="kpi-row">
        <div className="kpi">
          <div className="lbl">نام</div>
          <div className="val">{data.customer_name || 'نامشخص'}</div>
          <div className="sub">
            <span className={`badge ${data.is_existing ? 'ok' : 'warn'}`}>
              {data.is_existing ? 'مشتری موجود' : 'غیرمشتری · افر مشروط'}
            </span>
          </div>
        </div>
        <div className="kpi">
          <div className="lbl">سطح ریسک</div>
          <div className="val">
            <span className={`badge ${risk.cls}`}>{risk.text}</span>
          </div>
          <div className="sub">{data.risk_reason || 'ارزیابی RBCI'}</div>
        </div>
        <div className="kpi">
          <div className="lbl">امتیاز ریسک</div>
          <div className="val">{data.risk_score ?? '—'}</div>
          <div className="sub">۰ تا ۱۰۰ · کمتر بهتر است</div>
        </div>
        <div className="kpi">
          <div className="lbl">هدف مراجعه</div>
          <div className="val" style={{ fontSize: '1rem' }}>
            {data.visit_purpose || '—'}
          </div>
          <div className="sub">مؤثر بر رتبه‌بندی افرها</div>
        </div>
      </div>

      {!!data.notes_fa?.length && (
        <div className="card info">
          <h4>یادداشت‌های سیستمی</h4>
          <ul>
            {data.notes_fa.map((n) => (
              <li key={n}>{n}</li>
            ))}
          </ul>
        </div>
      )}

      {!!offers.length && (
        <>
          <div className="section-head">
            <h3>پیشنهادات ویژه</h3>
            <span className="count">{offers.length} مورد</span>
          </div>
          <div className="offers-grid">
            {offers.map((p) => (
              <ProductCard key={`offer-${p.product_id}`} product={p} kind="offer" />
            ))}
          </div>
        </>
      )}

      <div className="two-col">
        <div>
          <div className="section-head">
            <h3>محصولات مجاز</h3>
            <span className="count">{elig.length}</span>
          </div>
          {elig.length ? (
            elig.map((p) => <ProductCard key={`e-${p.product_id}`} product={p} kind="eligible" />)
          ) : (
            <div className="card info">
              <h4>موردی یافت نشد</h4>
              <p>هیچ محصول مجازی برای این پروفایل وجود ندارد.</p>
            </div>
          )}
        </div>
        <div>
          <div className="section-head">
            <h3>محصولات غیرمجاز</h3>
            <span className="count">{inelig.length}</span>
          </div>
          {inelig.length ? (
            inelig.map((p) => (
              <ProductCard key={`i-${p.product_id}`} product={p} kind="ineligible" />
            ))
          ) : (
            <div className="card eligible">
              <h4>همه محصولات مجاز</h4>
              <p>شکاف اهلیتی برای محصولات ثبت‌شده وجود ندارد.</p>
            </div>
          )}
        </div>
      </div>

      {data.default_warning && (
        <div className="card warn">
          <h4>هشدار پیامدهای عدم پرداخت</h4>
          <p>
            سطح ریسک فعلی: <b>{data.default_warning.current_risk_level}</b> ← احتمال تغییر به:{' '}
            <b>{data.default_warning.potential_risk_level}</b>
          </p>
          <ul>
            {(data.default_warning.consequences_fa || []).map((c) => (
              <li key={c}>{c}</li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}
