import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import { Download, Loader2 } from 'lucide-react'
import { api, type CodeShow } from '../api'
import { CopyButton } from '../components/CopyButton'

export function CodeShowView() {
  const { lang = '', hash = '' } = useParams()
  const [data, setData] = useState<CodeShow | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    setData(null)
    setError('')
    api
      .getCode(lang, hash)
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [lang, hash])

  const highlighted = useMemo(() => {
    if (!data) return ''
    if (hljs.getLanguage(data.language)) {
      return hljs.highlight(data.code, { language: data.language }).value
    }
    return hljs.highlightAuto(data.code).value
  }, [data])

  if (error) {
    return <p className="rounded-2xl bg-white px-4 py-10 text-center text-sm text-slate-500 shadow-md">{error}</p>
  }
  if (!data) {
    return (
      <p className="flex items-center justify-center gap-2 py-10 text-sm text-slate-500">
        <Loader2 size={16} className="animate-spin" /> 加载中…
      </p>
    )
  }

  const lineCount = data.code.split('\n').length

  return (
    <div className="overflow-hidden rounded-2xl bg-white shadow-md">
      <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3">
        <div className="flex items-center gap-2 text-sm">
          <span className="rounded-md bg-violet-100 px-2 py-0.5 text-xs font-medium text-violet-700">{data.language}</span>
          <span className="text-xs text-slate-400">{lineCount} 行</span>
        </div>
        <div className="flex items-center gap-2">
          <CopyButton text={data.code} label="复制代码" />
          <a
            href={data.downloadUrl}
            className="flex items-center gap-1 rounded-md border border-slate-300 bg-white px-2.5 py-1 text-xs text-slate-600 transition-colors hover:border-blue-400 hover:text-blue-600"
          >
            <Download size={14} /> 下载代码
          </a>
        </div>
      </div>

      <div className="code-text flex overflow-x-auto bg-slate-900 text-[13px] leading-6 text-slate-200">
        <div className="select-none border-r border-slate-700/50 px-3 py-3 text-right text-slate-500">
          {Array.from({ length: lineCount }, (_, i) => (
            <div key={i}>{i + 1}</div>
          ))}
        </div>
        <pre className="flex-1 px-4 py-3">
          <code className={`language-${data.language}`} dangerouslySetInnerHTML={{ __html: highlighted }} />
        </pre>
      </div>
    </div>
  )
}
