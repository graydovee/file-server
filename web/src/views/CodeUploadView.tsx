import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Loader2, Send } from 'lucide-react'
import { api, ApiError } from '../api'
import { LANGUAGE_EXTENSIONS } from '../utils/format'

export function CodeUploadView() {
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [language, setLanguage] = useState('go')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function submit() {
    if (!code.trim()) {
      setError('代码内容不能为空')
      return
    }
    setSubmitting(true)
    setError('')
    try {
      const resp = await api.uploadCode(code, language)
      navigate(resp.url)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '上传失败')
      setSubmitting(false)
    }
  }

  return (
    <div className="rounded-2xl bg-white p-6 shadow-md">
      <h2 className="text-base font-semibold text-slate-800">提交代码片段</h2>
      <p className="mt-1 text-sm text-slate-500">生成带语法高亮的分享链接</p>

      <div className="mt-4 space-y-3">
        <select
          value={language}
          onChange={(e) => setLanguage(e.target.value)}
          className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-blue-400"
        >
          {Object.keys(LANGUAGE_EXTENSIONS).map((lang) => (
            <option key={lang} value={lang}>
              {lang}
            </option>
          ))}
        </select>

        <textarea
          value={code}
          onChange={(e) => setCode(e.target.value)}
          rows={14}
          placeholder="在此粘贴代码…"
          className="code-text w-full resize-y rounded-xl border border-slate-200 bg-slate-50 p-3 text-[13px] leading-relaxed outline-none focus:border-blue-400"
        />

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button
          onClick={submit}
          disabled={submitting}
          className="flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm text-white shadow-sm transition-colors hover:bg-blue-700 disabled:opacity-60"
        >
          {submitting ? <Loader2 size={15} className="animate-spin" /> : <Send size={15} />}
          生成分享链接
        </button>
      </div>
    </div>
  )
}
