import { useEffect, useState } from 'react'
import { errorMessage, getOrders } from './api.ts'
import type { Order } from './api.ts'
import { formatReceivedAt } from './format.ts'

export default function OrdersList() {
  const [orders, setOrders] = useState<Order[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    let ignore = false
    getOrders().then(
      (list) => {
        if (ignore) return
        setOrders(list)
        setError(null)
        setLoading(false)
      },
      (err: unknown) => {
        if (ignore) return
        setError(errorMessage(err))
        setLoading(false)
      },
    )
    return () => {
      ignore = true
    }
  }, [attempt])

  function refresh() {
    setLoading(true)
    setAttempt((n) => n + 1)
  }

  return (
    <section className="orders">
      <div className="orders-toolbar">
        {orders !== null && orders.length > 0 && <span className="hint">Заявок: {orders.length}</span>}
        <button type="button" onClick={refresh} disabled={loading}>
          Обновить
        </button>
      </div>

      {error !== null && (
        <p className="error" role="alert">
          Не удалось загрузить заявки: {error}
        </p>
      )}

      {loading && orders === null && <p className="status">Загрузка…</p>}
      {orders !== null && orders.length === 0 && <p className="status">Заявок пока нет</p>}

      {/* Newest first: that is what the seller looks for after a new order. */}
      {orders !== null &&
        orders.toReversed().map((order) => <OrderCard key={order.id} order={order} />)}
    </section>
  )
}

function OrderCard({ order }: { order: Order }) {
  return (
    <article className="order" aria-labelledby={`order-${order.id}`}>
      <header className="order-header">
        <h2 id={`order-${order.id}`}>Заявка №{order.id}</h2>
        <time dateTime={order.received_at}>{formatReceivedAt(order.received_at)}</time>
        <span className="username">{order.username === '' ? '—' : `@${order.username}`}</span>
      </header>
      <dl className="answers">
        {order.answers.map((a) => (
          <div key={a.question_id} className="answer">
            <dt>{a.question_text}</dt>
            <dd>{a.answer}</dd>
          </div>
        ))}
      </dl>
    </article>
  )
}
