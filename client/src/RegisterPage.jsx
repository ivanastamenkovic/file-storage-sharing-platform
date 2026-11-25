import { useState } from 'react'
import { registerUser } from './api'
import { PASSWORD_RULES, isStrongPassword } from './passwordRules'
import { CheckIcon, MailIcon } from './icons'

const EMPTY = { firstName: '', lastName: '', email: '', password: '', confirmPassword: '' }

function validate(form) {
  const errors = {}
  if (!form.firstName.trim()) errors.firstName = 'First name is required'
  if (!form.lastName.trim()) errors.lastName = 'Last name is required'
  if (!form.email.trim()) errors.email = 'Email is required'
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim())) errors.email = 'Email is not valid'
  if (!form.password) errors.password = 'Password is required'
  else if (!isStrongPassword(form.password)) errors.password = 'Password is not strong enough'
  if (!form.confirmPassword) errors.confirmPassword = 'Please confirm your password'
  else if (form.password !== form.confirmPassword) errors.confirmPassword = 'Passwords do not match'
  return errors
}

function Field({ id, label, type = 'text', autoComplete, value, error, onChange, children }) {
  const [visible, setVisible] = useState(false)
  const isPassword = type === 'password'

  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <div className={isPassword ? 'input-wrap has-toggle' : 'input-wrap'}>
        <input
          id={id}
          name={id}
          type={isPassword && visible ? 'text' : type}
          autoComplete={autoComplete}
          value={value}
          onChange={onChange}
          aria-invalid={Boolean(error)}
          aria-describedby={error ? `${id}-error` : undefined}
          required
        />
        {isPassword && (
          <button
            type="button"
            className="toggle-visibility"
            onClick={() => setVisible((v) => !v)}
            aria-label={visible ? `Hide ${label.toLowerCase()}` : `Show ${label.toLowerCase()}`}
            aria-pressed={visible}
          >
            {visible ? 'Hide' : 'Show'}
          </button>
        )}
      </div>
      {children}
      {error && (
        <p id={`${id}-error`} className="field-error">
          {error}
        </p>
      )}
    </div>
  )
}

export default function RegisterPage() {
  const [form, setForm] = useState(EMPTY)
  const [errors, setErrors] = useState({})
  const [formError, setFormError] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const [registeredEmail, setRegisteredEmail] = useState(null)

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
      const res = await registerUser(form)
      setRegisteredEmail(res.email)
      setForm(EMPTY)
    } catch (err) {
      setErrors(err.fields ?? {})
      if (!err.fields || Object.keys(err.fields).length === 0) setFormError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  if (registeredEmail) {
    return (
      <section className="auth-card centered">
        <div className="status-icon info">
          <MailIcon size={28} />
        </div>
        <h2>Check your email</h2>
        <p role="status" className="auth-subtitle">
          We sent a confirmation link to <strong>{registeredEmail}</strong>. Open it to activate
          your account. The link is valid for 24 hours.
        </p>
      </section>
    )
  }

  return (
    <section className="auth-card">
      <h2>Create an account</h2>
      <p className="auth-subtitle">Store, organize and share your files in one place.</p>
      <form onSubmit={onSubmit} noValidate>
        <div className="field-row">
          <Field id="firstName" label="First name" autoComplete="given-name" value={form.firstName} error={errors.firstName} onChange={onChange} />
          <Field id="lastName" label="Last name" autoComplete="family-name" value={form.lastName} error={errors.lastName} onChange={onChange} />
        </div>
        <Field id="email" label="Email" type="email" autoComplete="email" value={form.email} error={errors.email} onChange={onChange} />
        <Field id="password" label="Password" type="password" autoComplete="new-password" value={form.password} error={errors.password} onChange={onChange}>
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
        <Field id="confirmPassword" label="Confirm password" type="password" autoComplete="new-password" value={form.confirmPassword} error={errors.confirmPassword} onChange={onChange} />

        {formError && (
          <p className="form-alert" role="alert">
            {formError}
          </p>
        )}

        <button type="submit" className="btn primary block" disabled={submitting}>
          {submitting && <span className="spinner" aria-hidden="true" />}
          {submitting ? 'Creating account...' : 'Create account'}
        </button>
      </form>
    </section>
  )
}
