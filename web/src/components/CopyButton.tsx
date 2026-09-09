import { useState } from 'react'
import { Check, Copy } from 'lucide-react'

export function CopyButton({ text, label = '复制' }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      // 非 HTTPS/旧浏览器的兜底方案
      const ta = document.createElement('textarea')
      ta.value = text
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <button
      onClick={copy}
      className="inline-flex shrink-0 items-center gap-1.5 border border-slate-300 bg-white px-2.5 py-1.5 text-xs text-slate-600 transition-colors duration-200 hover:border-indigo-600 hover:text-indigo-600"
    >
      {copied ? <Check size={14} className="text-emerald-600" /> : <Copy size={14} />}
      {copied ? '已复制' : label}
    </button>
  )
}

export function CodeLine({ code }: { code: string }) {
  return (
    <code className="font-mono block overflow-x-auto border border-indigo-100 bg-violet-50 p-3 text-[13px] leading-relaxed text-slate-800">
      {code}
    </code>
  )
}

/** 带标题的命令块：标题行 + 复制按钮 + 命令正文 */
export function CommandBlock({ label, code, multiline = false }: { label: string; code: string; multiline?: boolean }) {
  return (
    <div>
      <div className="mb-2 flex items-center justify-between gap-3">
        <h4 className="text-xs font-medium tracking-wide text-slate-500 uppercase">{label}</h4>
        <CopyButton text={code} />
      </div>
      {multiline ? (
        <pre className="font-mono overflow-x-auto border border-indigo-100 bg-violet-50 p-3 text-[13px] leading-relaxed whitespace-pre text-slate-800">
          {code}
        </pre>
      ) : (
        <CodeLine code={code} />
      )}
    </div>
  )
}
