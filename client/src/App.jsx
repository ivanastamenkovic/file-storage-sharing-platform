import NotificationsPage from './NotificationsPage'
import RegisterPage from './RegisterPage'
import ConfirmPage from './ConfirmPage'
import LoginPage from './LoginPage'
import ForgotPasswordPage from './ForgotPasswordPage'
import ResetPasswordPage from './ResetPasswordPage'
import ChangePasswordPage from './ChangePasswordPage'
import { CloudIcon } from './icons'
import { clearSession, getSession } from './session'

const CHANGE_PASSWORD_PATH = '/change-password'

const ROUTES = {
  '/register': RegisterPage,
  '/confirm': ConfirmPage,
  '/login': LoginPage,
  '/forgot-password': ForgotPasswordPage,
  '/reset-password': ResetPasswordPage,
  [CHANGE_PASSWORD_PATH]: ChangePasswordPage,
}

function handleSessionExpired() {
  clearSession()
  window.location.assign('/login')
}

export default function App() {
  const path = window.location.pathname
  const session = getSession()
  const passwordExpired = Boolean(session?.mustChangePassword)
  const Page = passwordExpired ? ChangePasswordPage : (ROUTES[path] ?? NotificationsPage)

  if (passwordExpired && path !== CHANGE_PASSWORD_PATH) {
    window.history.replaceState(null, '', CHANGE_PASSWORD_PATH)
  }

  const nav = passwordExpired
    ? []
    : session
    ? [
        { href: '/', label: 'Notifications' },
        { href: CHANGE_PASSWORD_PATH, label: 'Change password' },
      ]
    : [
        { href: '/login', label: 'Sign in' },
        { href: '/register', label: 'Register' },
      ]

  return (
    <>
      <header className="app-bar">
        <div className="app-bar-inner">
          <a href="/" className="brand">
            <span className="brand-mark">
              <CloudIcon />
            </span>
            File Storage and Sharing
          </a>
          <nav className="app-nav" aria-label="Main">
            {nav.map((item) => (
              <a key={item.href} href={item.href} aria-current={path === item.href ? 'page' : undefined}>
                {item.label}
              </a>
            ))}
            {session && (
              <>
                <span className="app-user" title={session.user.email}>
                  {session.user.firstName}
                </span>
                <button type="button" className="nav-button" onClick={handleSessionExpired}>
                  Sign out
                </button>
              </>
            )}
          </nav>
        </div>
      </header>
      <main className="container">
        <Page onSessionExpired={handleSessionExpired} />
      </main>
    </>
  )
}