import { useEffect, useRef, useState } from 'react'
import { confirmAccount } from './api'
import { AlertIcon, CheckIcon } from './icons'

export default function ConfirmPage() {
  const [token] = useState(() => new URLSearchParams(window.location.search).get('token'))
  const [state, setState] = useState(() =>
    token
      ? { status: 'loading', message: '' }
      : { status: 'error', message: 'This confirmation link is missing its token.' },
  )
  const started = useRef(false)

  useEffect(() => {
    // StrictMode runs effects twice in dev; the token can only be used once.
    if (started.current || !token) return
    started.current = true

    confirmAccount(token)
      .then((res) => setState({ status: 'success', message: res.message }))
      .catch((err) => setState({ status: 'error', message: err.message }))
  }, [token])

  return (
    <section className="auth-card centered" aria-live="polite">
      {state.status === 'loading' && (
        <>
          <div className="status-icon info">
            <span className="spinner large" aria-hidden="true" />
          </div>
          <h2>Confirming your account</h2>
          <p className="auth-subtitle">This only takes a moment.</p>
        </>
      )}
      {state.status === 'success' && (
        <>
          <div className="status-icon success">
            <CheckIcon size={28} />
          </div>
          <h2>Account confirmed</h2>
          <p role="status" className="auth-subtitle">{state.message}</p>
          <a href="/login" className="btn primary block">Sign in</a>
        </>
      )}
      {state.status === 'error' && (
        <>
          <div className="status-icon error">
            <AlertIcon size={28} />
          </div>
          <h2>Confirmation failed</h2>
          <p role="alert" className="auth-subtitle">{state.message}</p>
          <a href="/register" className="btn primary block">Back to registration</a>
        </>
      )}
    </section>
  )
}
