const KEY = 'session'

export function getSession() {
  try {
    const session = JSON.parse(localStorage.getItem(KEY))
    if (!session?.token || new Date(session.expiresAt) <= new Date()) return null
    return session
  } catch {
    return null
  }
}

export function saveSession(session) {
  try {
    localStorage.setItem(KEY, JSON.stringify(session))
  } catch {
    // storage unavailable; the session lasts only for this page
  }
}

export function clearSession() {
  try {
    localStorage.removeItem(KEY)
  } catch {
    // nothing to clear
  }
}
