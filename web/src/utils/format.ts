// 路径、大小格式化与命令拼装等纯函数

/** 与后端 code.go 的 extMap 保持一致 */
export const LANGUAGE_EXTENSIONS: Record<string, string> = {
  c: '.c',
  cpp: '.cpp',
  bash: '.sh',
  go: '.go',
  python: '.py',
  java: '.java',
  javascript: '.js',
  rust: '.rs',
  php: '.php',
  html: '.html',
  yaml: '.yaml',
  json: '.json',
  xml: '.xml',
}

/** 逐段 encodeURIComponent，保证路径中的中文/空格/特殊字符安全 */
export function encodePath(path: string): string {
  return path
    .split('/')
    .filter((s) => s.length > 0)
    .map((s) => encodeURIComponent(s))
    .join('/')
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 'B'
  for (const u of units) {
    if (value < 1024) break
    value /= 1024
    unit = u
  }
  return `${value.toFixed(value >= 100 ? 0 : 1)} ${unit}`
}

/** /download/2025/09 → { path: "2025/09", segments: ["2025", "09"] } */
export function parseBrowserPath(pathname: string): { path: string; segments: string[] } {
  const raw = pathname.startsWith('/download') ? pathname.slice('/download'.length) : pathname
  const segments = raw.split('/').filter((s) => s.length > 0).map(decodeURIComponent)
  return { path: segments.join('/'), segments }
}

/** 目录下文件在 code/<语言>/<文件名> 语义下的展示链接；非代码路径返回 null */
export function codeViewLink(dir: string, fileName: string): string | null {
  const parts = dir.split('/').filter((s) => s.length > 0)
  if (parts[0] !== 'code' || parts.length < 2) return null
  const lang = parts[1]
  if (!(lang in LANGUAGE_EXTENSIONS)) return null
  const dot = fileName.lastIndexOf('.')
  const base = dot > 0 ? fileName.slice(0, dot) : fileName
  return `/code/${encodeURIComponent(lang)}/${encodeURIComponent(base)}`
}

/** 浏览器当前 origin 下的批量下载脚本地址 */
export function buildScriptUrl(dir: string, os: 'bash' | 'powershell', download = false): string {
  const params = new URLSearchParams({ path: dir, os })
  if (download) params.set('download', '1')
  return `/script/download?${params.toString()}`
}

export function scriptCopyCommand(dir: string, os: 'bash' | 'powershell'): string {
  const url = `${window.location.origin}${buildScriptUrl(dir, os)}`
  return os === 'bash' ? `bash <(curl -fsSL '${url}')` : `irm '${url}' | iex`
}

export function archiveUrl(dir: string): string {
  return `/archive/${encodePath(dir)}`
}
