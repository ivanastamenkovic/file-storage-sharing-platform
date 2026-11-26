import { useEffect, useState } from 'react'
import { confirmPasswordReset } from './api'
import { PASSWORD_RULES, isStrongPassword } from './passwordRules'
import Field from './Field'
import { AlertIcon, CheckIcon } from './icons'

const REDIRECT_DELAY_MS = 3000
const LINK_ERRORS = ['INVALID_TOKEN', 'TOKEN_EXPIRED']
const EMPTY = { password: '', confirmPassword: '' }

function validate(form) {
  const errors = {}
  if (!form.password) errors.password = 'Password is required'
  else if (!isStrongPassword(form.password)) errors.password = 'Password is not strong enough'
  if (!form.confirmPassword) errors.confirmPassword = 'Please confirm your password'
  else if (form.password !== form.confirmPassword) errors.confirmPassword = 'Passwords do not match'
  return errors
}

export default function ResetPasswordPage() {
  const [token] = useState(() => new URLSearchParams(window.location.search).get('token'))
  const [form, setForm] = useState(EMPTY)
  const [errors, setErrors] = useState({})
  const [formError, setFormError] = useState(null)
  const [linkError, setLinkError] = useState(token ? null : 'This reset link is missing its token.')
  const [submitting, setSubmitting] = useState(false)
  const [doneMessage, setDoneMessage] = useState(null)

  useEffect(() => {
    if (!doneMessage) return undefined
    const timer = setTimeout(() => window.location.assign('/login'), REDIRECT_DELAY_MS)
    return () => clearTimeout(timer)
  }, [doneMessage])

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
      const res = await confirmPasswordReset(token, form.password, form.confirmPassword)
      setForm(EMPTY)
      setDoneMessage(res.message)
    } catch (err) {
      if (LINK_ERRORS.includes(err.code)) {
        setLinkError(err.message)
      } else if (err.fields && Object.keys(err.fields).length > 0) {
        setErrors(err.fields)
      } else {
        setFormError(err.message)
      }
    } finally {
      setSubmitting(false)
    }
  }

  if (doneMessage) {
    return (
      <section className="auth-card centered" aria-live="polite">
        <div className="status-icon success">
          <CheckIcon size={28} />
        </div>
        <h2>Password changed</h2>
        <p role="status" className="auth-subtitle">
          {doneMessage} Redirecting you to the sign in page...
        </p>
        <a href="/login" className="btn primary block">
          Sign in
        </a>
      </section>
    )
  }

  if (linkError) {
    return (
      <section className="auth-card centered" aria-live="polite">
        <div className="status-icon error">
          <AlertIcon size={28} />
        </div>
        <h2>Link not valid</h2>
        <p role="alert" className="auth-subtitle">
          {linkError}
        </p>
        <a href="/forgot-password" className="btn primary block">
          Request a new link
        </a>
      </section>
    )
  }

  return (
    <section className="auth-card">
      <h2>Choose a new password</h2>
      <p className="auth-subtitle">Pick a strong password you have not used before.</p>
      <form onSubmit={onSubmit} noValidate>
        <Field
          id="password"
          label="New password"
          type="password"
          autoComplete="new-password"
          value={form.password}
          error={errors.password}
          onChange={onChange}
        >
          <ul className="password-rules" aria-label="Password requirements">
            {PASSWORD_RULES.map((r) => (
              <li key={r.id} className={r.test(form.password) ? 'ok' : undefined}>
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