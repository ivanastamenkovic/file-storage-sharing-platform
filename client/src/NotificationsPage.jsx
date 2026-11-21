import { useCallback, useEffect, useState } from 'react'
import { fetchNotifications } from './api'

const TYPE_LABELS = {
  SHARED: 'Shared with you',
  NEW_FILE_IN_FOLDER: 'New file in folder',
  NEW_VERSION: 'New version',
}

export default function NotificationsPage() {
  const [notifications, setNotifications] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setNotifications(await fetchNotifications())
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  return (
    <section>
      <div className="section-header">
        <h2>Notifications</h2>
        <button onClick={load} disabled={loading}>
          Refresh
        </button>
      </div>

      {loading && <p className="muted">Loading notifications...</p>}

      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}

      {!loading && !error && notifications.length === 0 && (
        <p className="empty">You have no notifications yet.</p>
      )}

      <ul className="notification-list">
        {notifications.map((n) => (
          <li key={n.id} className={n.isRead ? 'notification' : 'notification unread'}>
            <div className="notification-type">{TYPE_LABELS[n.type] ?? n.type}</div>
            <div className="notification-message">{n.message}</div>
            <time className="notification-time" dateTime={n.createdAt}>
              {new Date(n.createdAt).toLocaleString()}
            </time>
          </li>
        ))}
      </ul>
    </section>
  )
}
