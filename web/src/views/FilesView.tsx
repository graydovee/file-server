import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  ChevronRight,
  Download,
  File as FileIcon,
  FileCode2,
  Folder,
  Home,
  Loader2,
  RefreshCw,
  Search,
  Terminal,
  Trash2,
} from 'lucide-react'
import { api, deleteFile, type FileEntry, type SearchResult } from '../api'
import { formatSize } from '../utils/format'
import {
  archiveUrl,
  buildScriptUrl,
  codeViewLink,
  encodePath,
  parseBrowserPath,
  scriptCopyCommand,
} from '../utils/format'
import { useServerInfo } from '../components/Layout'
import { CodeLine, CopyButton } from '../components/CopyButton'

type Os = 'bash' | 'powershell'

export function FilesView() {
  const { pathname } = useLocation()
  const info = useServerInfo()
  const { path, segments } = parseBrowserPath(pathname)

  const [entries, setEntries] = useState<FileEntry[]>([])
  const [nextCursor, setNextCursor] = useState('')
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState('')
  const [deleting, setDeleting] = useState('')

  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[] | null>(null)
  const [truncated, setTruncated] = useState(false)
  const [searching, setSearching] = useState(false)
  const [os, setOs] = useState<Os>('bash')
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()

  const fetchPage = useCallback(
    async (dir: string, cursor: string) => {
      cursor ? setLoadingMore(true) : setLoading(true)
      setError('')
      try {
        const page = await api.list(dir, cursor)
        setEntries((prev) => (cursor ? [...prev, ...page.entries] : page.entries))
        setNextCursor(page.nextCursor)
      } catch (e) {
        setError(e instanceof Error ? e.message : '加载失败')
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [],
  )

  useEffect(() => {
    setEntries([])
    setResults(null)
    setQuery('')
    fetchPage(path, '')
  }, [path, fetchPage])

  // 服务端搜索：防抖，带关键词时切换为结果视图
  useEffect(() => {
    const q = query.trim()
    if (!q) {
      setResults(null)
      setTruncated(false)
      return
    }
    setSearching(true)
    debounceRef.current = setTimeout(async () => {
      try {
        const resp = await api.search(path, q)
        setResults(resp.results)
        setTruncated(resp.truncated)
      } catch {
        setResults([])
      } finally {
        setSearching(false)
      }
    }, 350)
    return () => clearTimeout(debounceRef.current)
  }, [query, path])

  async function handleDelete(name: string) {
    const full = path ? `${path}/${name}` : name
    if (!window.confirm(`确认删除 ${full} ?`)) return
    setDeleting(full)
    try {
      await deleteFile(full)
      await fetchPage(path, '')
    } catch (e) {
      window.alert(e instanceof Error ? e.message : '删除失败')
    } finally {
      setDeleting('')
    }
  }

  const scriptCommand = scriptCopyCommand(path, os)
  const browsingCodeDir = path === 'code' || path.startsWith('code/')

  return (
    <div className="space-y-4">
      {/* 路径导航 */}
      <div className="flex flex-wrap items-center gap-1 rounded-2xl bg-white px-4 py-3 shadow-md">
        <Link
          to="/download"
          className="flex items-center gap-1 text-sm text-slate-500 hover:text-blue-600"
        >
          <Home size={15} /> 根目录
        </Link>
        {segments.map((seg, i) => {
          const target = '/download/' + encodePath(segments.slice(0, i + 1).join('/'))
          const isLast = i === segments.length - 1
          return (
            <span key={i} className="flex items-center gap-1">
              <ChevronRight size={14} className="text-slate-300" />
              {isLast ? (
                <span className="text-sm font-medium text-slate-800">{seg}</span>
              ) : (
                <Link to={target} className="text-sm text-slate-500 hover:text-blue-600">
                  {seg}
                </Link>
              )}
            </span>
          )
        })}
        <div className="ml-auto flex items-center gap-2">
          {loading ? (
            <Loader2 size={16} className="animate-spin text-slate-400" />
          ) : (
            <button
              onClick={() => fetchPage(path, '')}
              title="刷新"
              className="rounded-md p-1.5 text-slate-400 hover:bg-slate-100 hover:text-blue-600"
            >
              <RefreshCw size={15} />
            </button>
          )}
        </div>
      </div>

      {/* 搜索 + 批量下载 */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索当前目录下的文件（支持多个关键词）…"
            className="w-full rounded-xl border border-slate-200 bg-white py-2 pl-9 pr-9 text-sm shadow-sm outline-none transition-colors focus:border-blue-400"
          />
          {searching && <Loader2 size={15} className="absolute right-3 top-1/2 -translate-y-1/2 animate-spin text-slate-400" />}
        </div>
        <DownloadAll dir={path} presign={info?.presignEnabled ?? false} os={os} onOsChange={setOs} copyCommand={scriptCommand} />
      </div>

      {error && (
        <div className="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600">
          {error}
          <button onClick={() => fetchPage(path, '')} className="ml-2 underline">
            重试
          </button>
        </div>
      )}

      {/* 搜索结果视图 */}
      {results !== null ? (
        <div className="overflow-hidden rounded-2xl bg-white shadow-md">
          <div className="border-b border-slate-100 px-4 py-2.5 text-xs text-slate-500">
            找到 {results.length} 个结果
            {truncated && '，已达到返回上限，建议细化关键词'}
          </div>
          {results.length === 0 ? (
            <EmptyRow text="没有匹配的文件" />
          ) : (
            results.map((r) => (
              <a
                key={r.path}
                href={r.href ?? `/download/${encodePath(r.path)}`}
                className="flex items-center gap-3 border-b border-slate-50 px-4 py-2.5 text-sm last:border-0 hover:bg-slate-50"
              >
                <FileIcon size={16} className="shrink-0 text-slate-400" />
                <span className="code-text min-w-0 flex-1 truncate">{r.path}</span>
                <span className="shrink-0 text-xs text-slate-400">{formatSize(r.size)}</span>
              </a>
            ))
          )}
        </div>
      ) : (
        /* 目录浏览视图 */
        <div className="overflow-hidden rounded-2xl bg-white shadow-md">
          {loading ? (
            <EmptyRow text="加载中…" />
          ) : entries.length === 0 ? (
            <EmptyRow text="目录为空" />
          ) : (
            <>
              {entries.map((e) => (
                <Row
                  key={`${e.name}-${e.isDir}`}
                  dir={path}
                  entry={e}
                  deleting={deleting === (path ? `${path}/${e.name}` : e.name)}
                  showCodeLink={browsingCodeDir && !e.isDir}
                  onDelete={() => handleDelete(e.name)}
                />
              ))}
              {nextCursor && (
                <button
                  onClick={() => fetchPage(path, nextCursor)}
                  disabled={loadingMore}
                  className="flex w-full items-center justify-center gap-1.5 py-3 text-sm text-blue-600 hover:bg-slate-50 disabled:text-slate-400"
                >
                  {loadingMore && <Loader2 size={14} className="animate-spin" />}
                  加载更多
                </button>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}

function EmptyRow({ text }: { text: string }) {
  return <div className="px-4 py-10 text-center text-sm text-slate-400">{text}</div>
}

function Row({
  dir,
  entry,
  deleting,
  showCodeLink,
  onDelete,
}: {
  dir: string
  entry: FileEntry
  deleting: boolean
  showCodeLink: boolean
  onDelete: () => void
}) {
  const full = dir ? `${dir}/${entry.name}` : entry.name
  const codeLink = showCodeLink ? codeViewLink(dir, entry.name) : null

  return (
    <div className="flex items-center gap-3 border-b border-slate-50 px-4 py-2.5 text-sm last:border-0 hover:bg-slate-50">
      {entry.isDir ? (
        <Folder size={16} className="shrink-0 text-blue-400" />
      ) : (
        <FileIcon size={16} className="shrink-0 text-slate-400" />
      )}

      {entry.isDir ? (
        <Link to={`/download/${encodePath(full)}`} className="min-w-0 flex-1 truncate hover:text-blue-600">
          {entry.name}
        </Link>
      ) : (
        <a
          href={entry.href ?? `/download/${encodePath(full)}`}
          className="code-text min-w-0 flex-1 truncate hover:text-blue-600"
          title={entry.name}
        >
          {entry.name}
        </a>
      )}

      {codeLink && (
        <Link
          to={codeLink}
          className="flex shrink-0 items-center gap-1 rounded-md border border-violet-200 px-2 py-1 text-xs text-violet-600 hover:bg-violet-50"
        >
          <FileCode2 size={13} /> 查看代码
        </Link>
      )}

      {!entry.isDir && <span className="shrink-0 text-xs text-slate-400">{formatSize(entry.size)}</span>}

      <button
        onClick={onDelete}
        disabled={deleting}
        title="删除"
        className="shrink-0 rounded-md p-1.5 text-slate-300 transition-colors hover:bg-red-50 hover:text-red-500 disabled:opacity-50"
      >
        {deleting ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
      </button>
    </div>
  )
}

function DownloadAll({
  dir,
  presign,
  os,
  onOsChange,
  copyCommand,
}: {
  dir: string
  presign: boolean
  os: Os
  onOsChange: (os: Os) => void
  copyCommand: string
}) {
  const [showScript, setShowScript] = useState(false)

  if (presign) {
    return (
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex rounded-lg bg-slate-200/70 p-0.5">
          {(['bash', 'powershell'] as Os[]).map((o) => (
            <button
              key={o}
              onClick={() => onOsChange(o)}
              className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                os === o ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500'
              }`}
            >
              {o === 'bash' ? 'Linux/macOS' : 'Windows'}
            </button>
          ))}
        </div>
        <CopyButton text={copyCommand} label="复制下载全部命令" />
        <button
          onClick={() => setShowScript((v) => !v)}
          className="flex items-center gap-1 rounded-md border border-slate-300 bg-white px-2.5 py-1 text-xs text-slate-600 hover:border-blue-400 hover:text-blue-600"
        >
          <Terminal size={14} /> 下载脚本
        </button>
        {showScript && (
          <div className="w-full space-y-1.5">
            <a
              href={buildScriptUrl(dir, os, true)}
              className="inline-flex items-center gap-1 text-xs text-blue-600 underline"
            >
              <Download size={13} /> 下载 {os === 'bash' ? 'download.sh' : 'download.ps1'}
            </a>
            <CodeLine code={copyCommand} />
            <p className="text-xs text-slate-400">脚本内的直链有时效，请在有效期内执行；文件不经过服务器直接从存储下载。</p>
          </div>
        )}
      </div>
    )
  }

  return (
    <a
      href={archiveUrl(dir)}
      className="flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-1.5 text-xs text-white shadow-sm hover:bg-blue-700"
    >
      <Download size={14} /> 下载全部 (zip)
    </a>
  )
}
