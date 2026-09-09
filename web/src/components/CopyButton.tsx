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
      className="inline-flex shrink-0 items-center gap-1 rounded-md border border-slate-300 bg-white px-2.5 py-1 text-xs text-slate-600 transition-colors hover:border-blue-400 hover:text-blue-600"
    >
      {copied ? <Check size={14} className="text-green-600" /> : <Copy size={14} />}
      {copied ? '已复制' : label}
    </button>
  )
}

export function CodeLine({ code }: { code: string }) {
  return (
    <code className="code-text block overflow-x-auto rounded bg-slate-50 p-2.5 text-[13px] leading-relaxed text-slate-700">
      {code}
    </code>
  )
}
