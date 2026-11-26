import NotificationsPage from './NotificationsPage'
import RegisterPage from './RegisterPage'
import ConfirmPage from './ConfirmPage'
import LoginPage from './LoginPage'
import ForgotPasswordPage from './ForgotPasswordPage'
import ResetPasswordPage from './ResetPasswordPage'
import { CloudIcon } from './icons'
import { clearSession, getSession } from './session'

const ROUTES = {
  '/register': RegisterPage,
  '/confirm': ConfirmPage,
  '/login': LoginPage,
  '/forgot-password': ForgotPasswordPage,
  '/reset-password': ResetPasswordPage,
}

export default function App() {
  const path = window.location.pathname
  const Page = ROUTES[path] ?? NotificationsPage
  const session = getSession()

  const nav = session
    ? [{ href: '/', label: 'Notifications' }]
    : [
        { href: '/login', label: 'Sign in' },
        { href: '/register', label: 'Register' },
      ]

  const signOut = () => {
    clearSession()
    window.location.assign('/login')
  }

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
                <button type="button" className="nav-button" onClick={signOut}>
                  Sign out
                </button>
              </>
            )}
          </nav>
        </div>
      </header>
      <main className="container">
        <Page />
      </main>
    </>
  )
}
