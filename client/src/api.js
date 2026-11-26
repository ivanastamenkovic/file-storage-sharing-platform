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
  constructor(message, { status, code, fields, body } = {}) {
    super(message)
    this.status = status
    this.code = code
    this.fields = fields ?? {}
    this.body = body ?? null
  }
}

async function requestJSON(url, { method = 'GET', body, token } = {}, fallbackMessage) {
  const headers = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers.Authorization = `Bearer ${token}`

  let res
  try {
    res = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
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
      body: data,
    })
  }
  return data
}

function postJSON(url, body, fallbackMessage) {
  return requestJSON(url, { method: 'POST', body }, fallbackMessage)
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

export function fetchPasswordStatus(token) {
  return requestJSON('/api/users/password/status', { token }, 'Could not load your password details')
}

export function changePassword(token, form) {
  return requestJSON(
    '/api/users/password/change',
    { method: 'POST', body: form, token },
    'Could not change your password',
  )
}
