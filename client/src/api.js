export async function fetchNotifications() {
  const res = await fetch('/api/notifications/')
  if (!res.ok) {
    let message = 'Could not load notifications'
    try {
      const body = await res.json()
      if (body.message) message = body.message
    } catch {
      // response was not JSON, keep the default message
    }
    throw new Error(message)
  }
  return res.json()
}
