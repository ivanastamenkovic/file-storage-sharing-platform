import { useState } from 'react'

export default function Field({ id, label, type = 'text', autoComplete, value, error, onChange, children }) {
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