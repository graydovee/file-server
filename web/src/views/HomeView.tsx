import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowRight, Code2, FolderDown } from 'lucide-react'
import { CommandBlock } from '../components/CopyButton'
import { useServerInfo } from '../components/Layout'

type Platform = 'linux' | 'macos' | 'windows'

const PLATFORMS: { key: Platform; label: string }[] = [
  { key: 'linux', label: 'Linux' },
  { key: 'macos', label: 'macOS' },
  { key: 'windows', label: 'Windows' },
]

// 上传函数定义与旧版首页保持一致
const UPLOAD_FUNCTION = (uploadAddress: string) => `# 定义上传文件函数
upload_file() {
    local filename="$1"
    shift
    curl ${uploadAddress} \\
        --progress-bar \\
        -H "X-Filename: $(basename $filename)" \\
        -T "$filename" \\
        "$@" | cat
}`

export function HomeView() {
  const info = useServerInfo()
  const [platform, setPlatform] = useState<Platform>('linux')

  if (!info) {
    return <p className="py-16 text-center text-sm text-slate-500">正在连接服务器…</p>
  }

  const unix = platform === 'linux' || platform === 'macos'
  const uploadAddress = info.uploadAddress
  const downloadAddress = info.downloadAddress

  return (
    <div className="space-y-10">
      <section>
        <h1 className="font-display text-3xl font-bold tracking-tight text-slate-900">
          多设备之间
          <span className="text-indigo-600">传输文件</span>
        </h1>
        <p className="mt-3 max-w-xl text-sm leading-relaxed text-slate-500">
          无需公网 IP 与第三方下载工具，一条命令完成文件上传下载；浏览器可浏览、搜索与批量下载全部文件。
        </p>
      </section>

      <section className="grid gap-px border border-indigo-100 bg-indigo-100 sm:grid-cols-2">
        <Link
          to="/download"
          className="group flex items-center justify-between gap-4 bg-white p-6 transition-colors duration-200 hover:bg-indigo-600"
        >
          <span className="flex items-center gap-4">
            <FolderDown size={28} className="shrink-0 text-indigo-600 group-hover:text-white" />
            <span>
              <span className="font-display block text-base font-semibold text-slate-900 group-hover:text-white">
                文件下载
              </span>
              <span className="mt-1 block text-sm text-slate-500 group-hover:text-indigo-100">
                浏览、搜索、批量下载
              </span>
            </span>
          </span>
          <ArrowRight size={18} className="shrink-0 text-slate-300 transition-colors group-hover:text-white" />
        </Link>
        <Link
          to="/code"
          className="group flex items-center justify-between gap-4 bg-white p-6 transition-colors duration-200 hover:bg-indigo-600"
        >
          <span className="flex items-center gap-4">
            <Code2 size={28} className="shrink-0 text-indigo-600 group-hover:text-white" />
            <span>
              <span className="font-display block text-base font-semibold text-slate-900 group-hover:text-white">
                代码分享
              </span>
              <span className="mt-1 block text-sm text-slate-500 group-hover:text-indigo-100">
                粘贴代码，生成高亮分享链接
              </span>
            </span>
          </span>
          <ArrowRight size={18} className="shrink-0 text-slate-300 transition-colors group-hover:text-white" />
        </Link>
      </section>

      <section className="border border-indigo-100 bg-white">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-indigo-100 px-6 py-4">
          <h2 className="font-display text-base font-semibold text-slate-900">命令行上传 / 下载</h2>
          <div className="flex gap-5">
            {PLATFORMS.map(({ key, label }) => (
              <button
                key={key}
                onClick={() => setPlatform(key)}
                className={`py-1 text-sm transition-colors duration-200 ${
                  platform === key
                    ? 'font-medium text-indigo-600 underline decoration-indigo-600 decoration-2 underline-offset-8'
                    : 'text-slate-500 hover:text-slate-900'
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <div className="space-y-6 px-6 py-6">
          {unix ? (
            <>
              <CommandBlock label="定义上传函数（复制到终端执行一次）" code={UPLOAD_FUNCTION(uploadAddress)} multiline />
              <CommandBlock label="上传文件（之后随时可用）" code="upload_file [filename]" />
              <CommandBlock
                label="或：直接上传"
                code={
                  platform === 'linux'
                    ? `echo [filePath] | xargs -i curl -F "file=@{}" ${uploadAddress}`
                    : `echo [filePath] | xargs -I {} curl -F "file=@{}" ${uploadAddress}`
                }
              />
              <CommandBlock label="下载文件" code={`wget ${downloadAddress} -O [filePath]`} />
            </>
          ) : (
            <>
              <CommandBlock
                label="上传文件（PowerShell）"
                code={`curl.exe -T "[filePath]" -H "X-Filename: [filename]" ${uploadAddress}`}
              />
              <CommandBlock label="下载文件（PowerShell）" code={`Invoke-WebRequest -Uri ${downloadAddress} -OutFile [filePath]`} />
            </>
          )}
        </div>

        <p className="border-t border-indigo-100 bg-violet-50 px-6 py-3 text-xs leading-relaxed text-slate-500">
          上传成功后会返回可直接复制的下载命令；替换 [filename] / [filePath] 为实际路径。
          {platform === 'windows' && ' 注：旧版给出的 Invoke-WebRequest 上传命令缺少 X-Filename 会被服务端拒绝，已替换为 curl.exe（Windows 10+ 自带）。'}
        </p>
      </section>
    </div>
  )
}
