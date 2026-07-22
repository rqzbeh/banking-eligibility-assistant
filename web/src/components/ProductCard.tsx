import type { ProductMatch } from '../types'

function ScoreBar({ score }: { score: number }) {
  const s = Math.max(0, Math.min(100, score || 0))
  return (
    <div className="scorebar" title={`${s}`}>
      <span style={{ width: `${s}%` }} />
    </div>
  )
}

export function ProductCard({
  product,
  kind,
}: {
  product: ProductMatch
  kind: 'offer' | 'eligible' | 'ineligible'
}) {
  const title = product.product_name_fa || product.product_name || 'محصول'
  const reasons = (product.reasons_fa || product.reasons || []).join(' · ') || '—'
  const refs = (product.circular_refs || []).join('، ') || '—'
  const score = product.score || 0

  return (
    <article className={`card ${kind === 'offer' ? 'offer' : kind}`}>
      <h4>
        <span>{title}</span>
        {kind === 'eligible' && <span className="badge ok">مجاز</span>}
        {kind === 'ineligible' && <span className="badge bad">غیرمجاز</span>}
        {kind === 'offer' && <span className="badge accent">پیشنهاد ویژه</span>}
        {product.is_conditional && <span className="badge warn">افر مشروط</span>}
      </h4>

      <p style={{ margin: '0.25rem 0 0.4rem' }}>{reasons}</p>

      {(kind === 'offer' || kind === 'eligible') && (
        <>
          <ScoreBar score={score} />
          <div className="meta">
            امتیاز تناسب: <b>{score.toFixed(0)}</b> / ۱۰۰
          </div>
        </>
      )}

      {product.credit_limit_fa && (
        <div className="meta" style={{ marginTop: '0.5rem' }}>
          <b>سقف اعتبار:</b> {product.credit_limit_fa}
        </div>
      )}

      {!!product.conditions_fa?.length && (
        <>
          <div className="meta" style={{ marginTop: '0.45rem' }}>
            <b>شروط فعال‌سازی</b>
          </div>
          <ul>
            {product.conditions_fa.map((c) => (
              <li key={c}>{c}</li>
            ))}
          </ul>
        </>
      )}

      {!!product.obligations_fa?.length && (
        <>
          <div className="meta" style={{ marginTop: '0.45rem' }}>
            <b>تعهدات و الزامات</b>
          </div>
          <ul>
            {product.obligations_fa.map((o) => (
              <li key={o}>{o}</li>
            ))}
          </ul>
        </>
      )}

      {kind === 'ineligible' && (
        <>
          {(product.gaps || []).map((g, i) => (
            <div className="gap-item" key={`${g.field}-${i}`}>
              {g.description_fa || g.description}
              {g.advice_fa && <span className="tip">اقدام: {g.advice_fa}</span>}
            </div>
          ))}
          {!!product.advice_fa?.length && (
            <>
              <div className="meta" style={{ marginTop: '0.4rem' }}>
                <b>اقدامات پیشنهادی</b>
              </div>
              <ul>
                {product.advice_fa.map((a) => (
                  <li key={a}>{a}</li>
                ))}
              </ul>
            </>
          )}
          {!!product.alternatives_fa?.length && (
            <>
              <div className="meta" style={{ marginTop: '0.4rem' }}>
                <b>مسیرهای جایگزین</b>
              </div>
              <ul>
                {product.alternatives_fa.map((a) => (
                  <li key={a}>{a}</li>
                ))}
              </ul>
            </>
          )}
        </>
      )}

      <div className="meta">بخشنامه: {refs}</div>
    </article>
  )
}
