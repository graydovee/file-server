import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { FolderDown, Code2, Home } from 'lucide-react'
import { api, type ServerInfo } from '../api'

const InfoContext = createContext<ServerInfo | null>(null)

export function useServerInfo(): ServerInfo | null {
  return useContext(InfoContext)
}

export function Layout({ children }: { children: ReactNode }) {
  const [info, setInfo] = useState<ServerInfo | null>(null)

  useEffect(() => {
    api.getInfo().then(setInfo).catch(() => setInfo(null))
  }, [])

  const linkCls = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors ${
      isActive ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-200 hover:bg-white/10'
    }`

  return (
    <InfoContext.Provider value={info}>
      <div className="min-h-screen bg-gradient-to-b from-slate-800 via-slate-100 to-slate-100">
        <header className="bg-slate-800">
          <div className="mx-auto flex max-w-4xl items-center justify-between px-4 py-3">
            <NavLink to="/" className="text-lg font-semibold text-white">
              文件管理
            </NavLink>
            <nav className="flex items-center gap-1">
              <NavLink to="/" end className={linkCls}>
                <Home size={16} /> 首页
              </NavLink>
              <NavLink to="/download" className={linkCls}>
                <FolderDown size={16} /> 文件下载
              </NavLink>
              <NavLink to="/code" className={linkCls}>
                <Code2 size={16} /> 代码分享
              </NavLink>
            </nav>
          </div>
        </header>
        <main className="mx-auto max-w-4xl px-4 py-8">{children}</main>
      </div>
    </InfoContext.Provider>
  )
}
