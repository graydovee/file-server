import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Apple, FolderDown, Code2, Terminal } from 'lucide-react'
import { CodeLine, CopyButton } from '../components/CopyButton'
import { useServerInfo } from '../components/Layout'

type Platform = 'linux' | 'macos' | 'windows'

const PLATFORMS: { key: Platform; label: string; icon: typeof Terminal }[] = [
  { key: 'linux', label: 'Linux', icon: Terminal },
  { key: 'macos', label: 'macOS', icon: Apple },
  { key: 'windows', label: 'Windows', icon: Terminal },
]

export function HomeView() {
  const info = useServerInfo()
  const [platform, setPlatform] = useState<Platform>('linux')

  if (!info) {
    return <p className="text-center text-slate-500">正在连接服务器…</p>
  }

  const isWindows = platform === 'windows'
  const uploadCmd = isWindows
    ? `curl.exe -F "file=@本地文件" ${info.uploadAddress}`
    : `curl -F "file=@本地文件" ${info.uploadAddress}`
  const downloadCmd = isWindows
    ? `Invoke-WebRequest -Uri ${info.downloadAddress} -OutFile 保存文件名`
    : `wget ${info.downloadAddress} -O 保存文件名`

  return (
    <div className="space-y-6">
      <section className="grid gap-4 sm:grid-cols-2">
        <Link
          to="/download"
          className="group flex items-center gap-4 rounded-2xl bg-blue-600 p-6 text-white shadow-lg shadow-blue-600/20 transition-transform hover:-translate-y-0.5"
        >
          <FolderDown size={36} className="shrink-0" />
          <div>
            <h2 className="text-lg font-semibold">文件下载</h2>
            <p className="mt-1 text-sm text-blue-100">浏览服务器文件、搜索、批量下载</p>
          </div>
        </Link>
        <Link
          to="/code"
          className="group flex items-center gap-4 rounded-2xl bg-white p-6 shadow-md transition-transform hover:-translate-y-0.5"
        >
          <Code2 size={36} className="shrink-0 text-violet-600" />
          <div>
            <h2 className="text-lg font-semibold text-slate-800">代码分享</h2>
            <p className="mt-1 text-sm text-slate-500">粘贴代码片段，生成高亮分享链接</p>
          </div>
        </Link>
      </section>

      <section className="rounded-2xl bg-white p-6 shadow-md">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-base font-semibold text-slate-800">命令行上传 / 下载</h2>
          <div className="flex rounded-lg bg-slate-100 p-1">
            {PLATFORMS.map(({ key, label, icon: Icon }) => (
              <button
                key={key}
                onClick={() => setPlatform(key)}
                className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors ${
                  platform === key ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500 hover:text-slate-700'
                }`}
              >
                <Icon size={15} />
                {label}
              </button>
            ))}
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <div className="mb-1.5 flex items-center justify-between">
              <h3 className="text-sm font-medium text-slate-600">上传文件</h3>
              <CopyButton text={uploadCmd} />
            </div>
            <CodeLine code={uploadCmd} />
          </div>
          <div>
            <div className="mb-1.5 flex items-center justify-between">
              <h3 className="text-sm font-medium text-slate-600">下载文件</h3>
              <CopyButton text={downloadCmd} />
            </div>
            <CodeLine code={downloadCmd} />
          </div>
        </div>

        <p className="mt-4 text-xs text-slate-400">
          上传成功后会返回可直接复制的下载命令；替换命令中的"本地文件 / 保存文件名"为实际文件名。
        </p>
      </section>
    </div>
  )
}
