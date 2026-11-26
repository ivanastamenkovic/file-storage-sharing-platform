import { useState } from 'react'
import { login, verifyLoginCode } from './api'
import { saveSession } from './session'
import { MailIcon } from './icons'

function TextField({ id, label, error, ...props }) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <div className="input-wrap">
        <input
          id={id}
          name={id}
          aria-invalid={Boolean(error)}
          aria-describedby={error ? `${id}-error` : undefined}
          required
          {...props}
        />
      </div>
      {error && (
        <p id={`${id}-error`} className="field-error">
          {error}
        </p>
      )}
    </div>
  )
}

function CredentialsStep({ onChallenge }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState(null)
  const [submitting, setSubmitting] = useState(false)

  const onSubmit = async (e) => {
    e.preventDefault()
    if (!email.trim() || !password) {
      setError({ message: 'Enter your email and password' })
      return
    }
    setError(null)
    setSubmitting(true)
    try {
      const res = await login(email.trim(), password)
      onChallenge({ ...res, email: email.trim().toLowerCase() })
    } catch (err) {
      setError({ message: err.message, code: err.code })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <h2>Sign in</h2>
      <p className="auth-subtitle">Enter your email and password. We will email you a one-time code.</p>
      <form onSubmit={onSubmit} noValidate>
        <TextField
          id="email"
          label="Email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <div className="field">
          <label htmlFor="password">Password</label>
          <div className="input-wrap has-toggle">
            <input
              id="password"
              name="password"
              type={showPassword ? 'text' : 'password'}
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
            <button
              type="button"
              className="toggle-visibility"
              onClick={() => setShowPassword((v) => !v)}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
              aria-pressed={showPassword}
            >
              {showPassword ? 'Hide' : 'Show'}
            </button>
          </div>
        </div>

        {error && (
          <p className="form-alert" role="alert">
            {error.message}
          </p>
        )}

        <button type="submit" className="btn primary block" disabled={submitting}>
          {submitting && <span className="spinner" aria-hidden="true" />}
          {submitting ? 'Checking...' : 'Continue'}
        </button>
      </form>
      <p className="auth-footer">
        No account yet? <a href="/register">Create one</a>
      </p>
    </>
  )
}

function CodeStep({ challenge, onBack }) {
  const [otp, setOtp] = useState('')
  const [error, setError] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const [finished, setFinished] = useState(false)

  const onSubmit = async (e) => {
    e.preventDefault()
    if (!/^\d{6}$/.test(otp)) {
      setError('Enter the 6-digit code from the email')
      return
    }
    setError(null)
    setSubmitting(true)
    try {
      const session = await verifyLoginCode(challenge.challengeId, otp)
      saveSession(session)
      window.location.assign('/')
    } catch (err) {
      // An expired, used up or locked code cannot be retried; a new sign-in is needed.
      if (['OTP_EXPIRED', 'TOO_MANY_ATTEMPTS'].includes(err.code)) setFinished(true)
      setError(err.message)
      setOtp('')
      setSubmitting(false)
    }
  }

  return (
    <>
      <div className="status-icon info">
        <MailIcon size={28} />
      </div>
      <h2 className="centered-text">Check your email</h2>
      <p className="auth-subtitle centered-text">
        We sent a 6-digit code to <strong>{challenge.email}</strong>. It expires in{' '}
        {Math.round(challenge.expiresInSeconds / 60)} minutes.
      </p>
      <form onSubmit={onSubmit} noValidate>
        <div className="field">
          <label htmlFor="otp">One-time code</label>
          <input
            id="otp"
            name="otp"
            className="otp-input"
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            placeholder="000000"
            value={otp}
            onChange={(e) => setOtp(e.target.value.replace(/\D/g, '').slice(0, 6))}
            aria-invalid={Boolean(error)}
            disabled={finished}
            autoFocus
          />
        </div>

        {error && (
          <p className="form-alert" role="alert">
            {error}
          </p>
        )}

        {finished ? (
          <button type="button" className="btn primary block" onClick={onBack}>
            Sign in again
          </button>
        ) : (
          <button type="submit" className="btn primary block" disabled={submitting}>
            {submitting && <span className="spinner" aria-hidden="true" />}
            {submitting ? 'Verifying...' : 'Verify and sign in'}
          </button>
        )}
      </form>
      <p className="auth-footer">
        Didn&apos;t get it?{' '}
        <button type="button" className="link-button" onClick={onBack}>
          Send a new code
        </button>
      </p>
    </>
  )
}

export default function LoginPage() {
  const [challenge, setChallenge] = useState(null)

  return (
    <section className="auth-card">
      {challenge ? (
        <CodeStep challenge={challenge} onBack={() => setChallenge(null)} />
      ) : (
        <CredentialsStep onChallenge={setChallenge} />
      )}
    </section>
  )
}
