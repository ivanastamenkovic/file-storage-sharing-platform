import NotificationsPage from './NotificationsPage'
import RegisterPage from './RegisterPage'
import ConfirmPage from './ConfirmPage'
import { CloudIcon } from './icons'

const ROUTES = {
  '/register': RegisterPage,
  '/confirm': ConfirmPage,
}

const NAV = [
  { href: '/', label: 'Notifications' },
  { href: '/register', label: 'Register' },
]

export default function App() {
  const path = window.location.pathname
  const Page = ROUTES[path] ?? NotificationsPage

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
            {NAV.map((item) => (
              <a key={item.href} href={item.href} aria-current={path === item.href ? 'page' : undefined}>
                {item.label}
              </a>
            ))}
          </nav>
        </div>
      </header>
      <main className="container">
        <Page />
      </main>
    </>
  )
}
