import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { api, type ServerInfo } from '../api'

const InfoContext = createContext<ServerInfo | null>(null)

export function useServerInfo(): ServerInfo | null {
  return useContext(InfoContext)
}

const NAV = [
  { to: '/', label: '首页', end: true },
  { to: '/download', label: '文件下载', end: false },
  { to: '/code', label: '代码分享', end: false },
]

export function Layout({ children }: { children: ReactNode }) {
  const [info, setInfo] = useState<ServerInfo | null>(null)

  useEffect(() => {
    api.getInfo().then(setInfo).catch(() => setInfo(null))
  }, [])

  const linkCls = ({ isActive }: { isActive: boolean }) =>
    `px-1 py-2 text-sm transition-colors duration-200 ${
      isActive
        ? 'font-medium text-indigo-600 underline decoration-indigo-600 decoration-2 underline-offset-8'
        : 'text-slate-500 hover:text-slate-900'
    }`

  return (
    <InfoContext.Provider value={info}>
      <div className="min-h-screen bg-[#F5F3FF]">
        <header className="border-b border-indigo-100 bg-white">
          <div className="mx-auto flex h-16 max-w-4xl items-center justify-between px-4">
            <NavLink to="/" className="flex items-center gap-2.5">
              <span aria-hidden className="h-3 w-3 bg-indigo-600" />
              <span className="font-display text-lg font-bold tracking-tight text-slate-900">fileserver</span>
            </NavLink>
            <nav className="flex items-center gap-6">
              {NAV.map(({ to, label, end }) => (
                <NavLink key={to} to={to} end={end} className={linkCls}>
                  {label}
                </NavLink>
              ))}
            </nav>
          </div>
        </header>
        <main className="mx-auto max-w-4xl px-4 py-10">{children}</main>
      </div>
    </InfoContext.Provider>
  )
}
