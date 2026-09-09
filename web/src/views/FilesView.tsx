import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  ChevronRight,
  Download,
  File as FileIcon,
  FileCode2,
  Folder,
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

const CARD = 'border border-indigo-100 bg-white'
const BTN_PRIMARY =
  'flex items-center gap-1.5 border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white transition-colors duration-200 hover:bg-indigo-800'
const BTN_SECONDARY =
  'flex items-center gap-1.5 border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-600 transition-colors duration-200 hover:border-indigo-600 hover:text-indigo-600'

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

  const fetchPage = useCallback(async (dir: string, cursor: string) => {
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
  }, [])

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
    <div className="space-y-5">
      {/* 路径导航 */}
      <div className={`flex flex-wrap items-center gap-1 px-4 py-3 ${CARD}`}>
        <Link to="/download" className="text-sm text-slate-500 transition-colors hover:text-indigo-600">
          根目录
        </Link>
        {segments.map((seg, i) => {
          const target = '/download/' + encodePath(segments.slice(0, i + 1).join('/'))
          const isLast = i === segments.length - 1
          return (
            <span key={i} className="flex items-center gap-1">
              <ChevronRight size={14} className="text-indigo-200" />
              {isLast ? (
                <span className="font-medium text-sm text-slate-900">{seg}</span>
              ) : (
                <Link to={target} className="text-sm text-slate-500 transition-colors hover:text-indigo-600">
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
              className="p-1.5 text-slate-400 transition-colors hover:text-indigo-600"
            >
              <RefreshCw size={15} />
            </button>
          )}
        </div>
      </div>

      {/* 搜索 + 批量下载 */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search size={15} className="absolute top-1/2 left-3 -translate-y-1/2 text-slate-400" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索当前目录下的文件（支持多个关键词）…"
            className="w-full border border-indigo-100 bg-white py-2 pr-9 pl-9 text-sm text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-indigo-600"
          />
          {searching && (
            <Loader2 size={15} className="absolute top-1/2 right-3 -translate-y-1/2 animate-spin text-slate-400" />
          )}
        </div>
        <DownloadAll
          dir={path}
          presign={info?.presignEnabled ?? false}
          os={os}
          onOsChange={setOs}
          copyCommand={scriptCommand}
        />
      </div>

      {error && (
        <div className="flex items-center gap-3 border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
          <button onClick={() => fetchPage(path, '')} className="underline hover:text-red-900">
            重试
          </button>
        </div>
      )}

      {/* 搜索结果视图 */}
      {results !== null ? (
        <div className={CARD}>
          <div className="border-b border-indigo-100 bg-violet-50 px-4 py-2.5 text-xs text-slate-500">
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
                className="flex items-center gap-3 border-b border-indigo-50 px-4 py-2.5 text-sm transition-colors last:border-0 hover:bg-violet-50"
              >
                <FileIcon size={16} className="shrink-0 text-slate-400" />
                <span className="font-mono min-w-0 flex-1 truncate text-slate-800">{r.path}</span>
                <span className="shrink-0 text-xs text-slate-400">{formatSize(r.size)}</span>
              </a>
            ))
          )}
        </div>
      ) : (
        /* 目录浏览视图 */
        <div className={CARD}>
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
                  className="w-full border-t border-indigo-100 py-3 text-sm font-medium text-indigo-600 transition-colors hover:bg-violet-50 disabled:text-slate-400"
                >
                  {loadingMore ? '加载中…' : '加载更多'}
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
    <div className="flex items-center gap-3 border-b border-indigo-50 px-4 py-2.5 text-sm transition-colors last:border-0 hover:bg-violet-50">
      {entry.isDir ? (
        <Folder size={16} className="shrink-0 text-indigo-500" />
      ) : (
        <FileIcon size={16} className="shrink-0 text-slate-400" />
      )}

      {entry.isDir ? (
        <Link to={`/download/${encodePath(full)}`} className="min-w-0 flex-1 truncate transition-colors hover:text-indigo-600">
          {entry.name}
        </Link>
      ) : (
        <a
          href={entry.href ?? `/download/${encodePath(full)}`}
          className="font-mono min-w-0 flex-1 truncate text-slate-800 transition-colors hover:text-indigo-600"
          title={entry.name}
        >
          {entry.name}
        </a>
      )}

      {codeLink && (
        <Link
          to={codeLink}
          className="flex shrink-0 items-center gap-1 border border-indigo-200 px-2 py-1 text-xs text-indigo-600 transition-colors hover:bg-indigo-600 hover:text-white"
        >
          <FileCode2 size={13} /> 查看代码
        </Link>
      )}

      {!entry.isDir && <span className="shrink-0 text-xs text-slate-400">{formatSize(entry.size)}</span>}

      <button
        onClick={onDelete}
        disabled={deleting}
        title="删除"
        className="shrink-0 p-1.5 text-slate-300 transition-colors hover:text-red-600 disabled:opacity-50"
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
        <div className="flex border border-indigo-100 bg-white">
          {(['bash', 'powershell'] as Os[]).map((o) => (
            <button
              key={o}
              onClick={() => onOsChange(o)}
              className={`px-2.5 py-1.5 text-xs transition-colors duration-200 ${
                os === o ? 'bg-indigo-600 text-white' : 'text-slate-500 hover:text-slate-900'
              }`}
            >
              {o === 'bash' ? 'Linux/macOS' : 'Windows'}
            </button>
          ))}
        </div>
        <CopyButton text={copyCommand} label="复制下载全部命令" />
        <button onClick={() => setShowScript((v) => !v)} className={BTN_SECONDARY}>
          <Terminal size={14} /> 下载脚本
        </button>
        {showScript && (
          <div className="w-full space-y-2">
            <a
              href={buildScriptUrl(dir, os, true)}
              className="inline-flex items-center gap-1 text-xs text-indigo-600 underline hover:text-indigo-800"
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
    <a href={archiveUrl(dir)} className={BTN_PRIMARY}>
      <Download size={14} /> 下载全部 (zip)
    </a>
  )
}
