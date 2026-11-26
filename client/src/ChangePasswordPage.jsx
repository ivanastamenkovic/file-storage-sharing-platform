import { useEffect, useState } from 'react'
import { changePassword, fetchPasswordStatus } from './api'
import { PASSWORD_RULES, isStrongPassword } from './passwordRules'
import { clearSession, getSession, saveSession } from './session'
import Field from './Field'
import { AlertIcon, CheckIcon } from './icons'

const EMPTY = { currentPassword: '', newPassword: '', confirmPassword: '' }

function validate(form) {
  const errors = {}
  if (!form.currentPassword) errors.currentPassword = 'Current password is required'
  if (!form.newPassword) errors.newPassword = 'New password is required'
  else if (!isStrongPassword(form.newPassword)) errors.newPassword = 'Password is not strong enough'
  else if (form.newPassword === form.currentPassword)
    errors.newPassword = 'New password must be different from the current one'
  if (!form.confirmPassword) errors.confirmPassword = 'Please confirm your new password'
  else if (form.newPassword !== form.confirmPassword) errors.confirmPassword = 'Passwords do not match'
  return errors
}

function formatDate(value) {
  return new Date(value).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

function withLock(status) {
  return { ...status, locked: new Date(status.canChangeAt) > new Date() }
}

function signInAgain() {
  clearSession()
  window.location.assign('/login')
}

export default function ChangePasswordPage() {
  const [session] = useState(getSession)
  const [status, setStatus] = useState(null)
  const [loadError, setLoadError] = useState(null)
  const [form, setForm] = useState(EMPTY)
  const [errors, setErrors] = useState({})
  const [formError, setFormError] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const [doneMessage, setDoneMessage] = useState(null)

  useEffect(() => {
    if (!session) {
      window.location.assign('/login')
      return
    }
    fetchPasswordStatus(session.token)
      .then((s) => setStatus(withLock(s)))
      .catch((err) => {
        if (err.status === 401) signInAgain()
        else setLoadError(err.message)
      })
  }, [session])

  const onChange = (e) => {
    const { name, value } = e.target
    setForm((f) => ({ ...f, [name]: value }))
    setErrors((errs) => ({ ...errs, [name]: undefined }))
  }

  const onSubmit = async (e) => {
    e.preventDefault()
    setFormError(null)
    const clientErrors = validate(form)
    setErrors(clientErrors)
    if (Object.keys(clientErrors).length > 0) return

    setSubmitting(true)
    try {
      const res = await changePassword(session.token, form)
      saveSession({ ...session, mustChangePassword: false })
      setForm(EMPTY)
      setStatus(withLock(res.status))
      setDoneMessage(res.message)
    } catch (err) {
      if (err.status === 401) {
        signInAgain()
        return
      }
      if (err.code === 'PASSWORD_TOO_RECENT') {
        fetchPasswordStatus(session.token).then((s) => setStatus(withLock(s)), () => {})
      }
      if (Object.keys(err.fields).length > 0) setErrors(err.fields)
      else setFormError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  if (!session) return null

  if (doneMessage) {
    return (
      <section className="auth-card centered" aria-live="polite">
        <div className="status-icon success">
          <CheckIcon size={28} />
        </div>
        <h2>Password changed</h2>
        <p role="status" className="auth-subtitle">
          {doneMessage}
        </p>
        <a href="/" className="btn primary block">
          Continue
        </a>
      </section>
    )
  }

  if (loadError) {
    return (
      <section className="auth-card centered">
        <div className="status-icon error">
          <AlertIcon size={28} />
        </div>
        <h2>Something went wrong</h2>
        <p role="alert" className="auth-subtitle">
          {loadError}
        </p>
        <button type="button" className="btn primary block" onClick={() => window.location.reload()}>
          Try again
        </button>
      </section>
    )
  }

  if (!status) {
    return (
      <section className="auth-card centered" aria-busy="true">
        <span className="spinner large" aria-hidden="true" />
        <p className="auth-subtitle">Loading...</p>
      </section>
    )
  }

  if (status.locked) {
    return (
      <section className="auth-card centered">
        <div className="status-icon info">
          <AlertIcon size={28} />
        </div>
        <h2>Password changed recently</h2>
        <p className="auth-subtitle">
          Your password was last changed on {formatDate(status.changedAt)}. You can change it again after{' '}
          <strong>{formatDate(status.canChangeAt)}</strong>.
        </p>
        <a href="/" className="btn primary block">
          Back
        </a>
      </section>
    )
  }

  return (
    <section className="auth-card">
      <h2>Change password</h2>
      {status.mustChange ? (
        <p className="form-alert" role="alert">
          Your password has expired. Choose a new one to continue.
        </p>
      ) : (
        <p className="auth-subtitle">
          Last changed on {formatDate(status.changedAt)}.
          {status.expiresAt && <> It expires on {formatDate(status.expiresAt)}.</>}
        </p>
      )}
      <form onSubmit={onSubmit} noValidate>
        <Field
          id="currentPassword"
          label="Current password"
          type="password"
          autoComplete="current-password"
          value={form.currentPassword}
          error={errors.currentPassword}
          onChange={onChange}
        />
        <Field
          id="newPassword"
          label="New password"
          type="password"
          autoComplete="new-password"
          value={form.newPassword}
          error={errors.newPassword}
          onChange={onChange}
        >
          <ul className="password-rules" aria-label="Password requirements">
            {PASSWORD_RULES.map((r) => (
              <li key={r.id} className={r.test(form.newPassword) ? 'ok' : undefined}>
                <span className="rule-dot">
                  <CheckIcon size={10} />
                </span>
                {r.label}
              </li>
            ))}
          </ul>
        </Field>
        <Field
          id="confirmPassword"
          label="Confirm new password"
          type="password"
          autoComplete="new-password"
          value={form.confirmPassword}
          error={errors.confirmPassword}
          onChange={onChange}
        />

        {formError && (
          <p className="form-alert" role="alert">
            {formError}
          </p>
        )}

        <button type="submit" className="btn primary block" disabled={submitting}>
          {submitting && <span className="spinner" aria-hidden="true" />}
          {submitting ? 'Saving...' : 'Change password'}
        </button>
      </form>
    </section>
  )
}
