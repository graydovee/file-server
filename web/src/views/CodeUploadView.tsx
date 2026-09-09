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
    <div className="border border-indigo-100 bg-white">
      <div className="border-b border-indigo-100 px-6 py-4">
        <h2 className="font-display text-base font-semibold text-slate-900">提交代码片段</h2>
        <p className="mt-1 text-sm text-slate-500">生成带语法高亮的分享链接</p>
      </div>

      <div className="space-y-4 px-6 py-6">
        <label className="block">
          <span className="mb-2 block text-xs font-medium tracking-wide text-slate-500 uppercase">语言</span>
          <select
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            className="border border-indigo-100 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition-colors focus:border-indigo-600"
          >
            {Object.keys(LANGUAGE_EXTENSIONS).map((lang) => (
              <option key={lang} value={lang}>
                {lang}
              </option>
            ))}
          </select>
        </label>

        <label className="block">
          <span className="mb-2 block text-xs font-medium tracking-wide text-slate-500 uppercase">代码内容</span>
          <textarea
            value={code}
            onChange={(e) => setCode(e.target.value)}
            rows={14}
            placeholder="在此粘贴代码…"
            className="font-mono w-full resize-y border border-indigo-100 bg-violet-50 p-3 text-[13px] leading-relaxed text-slate-800 outline-none transition-colors placeholder:text-slate-400 focus:border-indigo-600 focus:bg-white"
          />
        </label>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button
          onClick={submit}
          disabled={submitting}
          className="flex items-center gap-1.5 border border-indigo-600 bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white transition-colors duration-200 hover:bg-indigo-800 disabled:opacity-60"
        >
          {submitting ? <Loader2 size={15} className="animate-spin" /> : <Send size={15} />}
          生成分享链接
        </button>
      </div>
    </div>
  )
}
