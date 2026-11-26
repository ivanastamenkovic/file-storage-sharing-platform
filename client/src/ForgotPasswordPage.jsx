import { useState } from 'react'
import { requestPasswordReset } from './api'
import Field from './Field'
import { MailIcon } from './icons'

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState(null)

  const onSubmit = async (e) => {
    e.preventDefault()
    const trimmed = email.trim()
    if (!trimmed) {
      setError('Email is required')
      return
    }
    if (!EMAIL_PATTERN.test(trimmed)) {
      setError('Email is not valid')
      return
    }

    setError(null)
    setSubmitting(true)
    try {
      const res = await requestPasswordReset(trimmed)
      setMessage(res.message)
    } catch (err) {
      setError(err.fields?.email ?? err.message)
    } finally {
      setSubmitting(false)
    }
  }

  if (message) {
    return (
      <section className="auth-card centered">
        <div className="status-icon info">
          <MailIcon size={28} />
        </div>
        <h2>Check your email</h2>
        <p role="status" className="auth-subtitle">
          {message}
        </p>
        <a href="/login" className="btn primary block">
          Back to sign in
        </a>
      </section>
    )
  }

  return (
    <section className="auth-card">
      <h2>Forgot your password?</h2>
      <p className="auth-subtitle">Enter your email and we will send you a link to choose a new password.</p>
      <form onSubmit={onSubmit} noValidate>
        <Field
          id="email"
          label="Email"
          type="email"
          autoComplete="email"
          value={email}
          error={error}
          onChange={(e) => {
            setEmail(e.target.value)
            setError(null)
          }}
        />
        <button type="submit" className="btn primary block" disabled={submitting}>
          {submitting && <span className="spinner" aria-hidden="true" />}
          {submitting ? 'Sending...' : 'Send reset link'}
        </button>
      </form>
      <p className="auth-footer">
        <a href="/login">Back to sign in</a>
      </p>
    </section>
  )
}