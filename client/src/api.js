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

export class ApiError extends Error {
  constructor(message, { status, code, fields } = {}) {
    super(message)
    this.status = status
    this.code = code
    this.fields = fields ?? {}
  }
}

async function postJSON(url, body, fallbackMessage) {
  let res
  try {
    res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  } catch {
    throw new ApiError('Could not reach the server, check your connection')
  }

  let data = null
  try {
    data = await res.json()
  } catch {
    // response was not JSON
  }

  if (!res.ok) {
    throw new ApiError(data?.message || fallbackMessage, {
      status: res.status,
      code: data?.error,
      fields: data?.fields,
    })
  }
  return data
}

export function registerUser(form) {
  return postJSON('/api/users/register', form, 'Registration failed')
}

export function confirmAccount(token) {
  return postJSON('/api/users/confirm', { token }, 'Could not confirm your account')
}

export function login(email, password) {
  return postJSON('/api/users/login', { email, password }, 'Could not sign in')
}

export function verifyLoginCode(challengeId, otp) {
  return postJSON('/api/users/login/verify', { challengeId, otp }, 'Could not verify the code')
}

export function requestPasswordReset(email) {
  return postJSON('/api/users/password-reset/request', { email }, 'Could not send the reset link')
}

export function confirmPasswordReset(token, password, confirmPassword) {
  return postJSON(
    '/api/users/password-reset/confirm',
    { token, password, confirmPassword },
    'Could not reset your password',
  )
}
