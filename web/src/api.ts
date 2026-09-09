// 与后端 /api 系列 JSON 接口对应的类型与请求封装
import { encodePath } from './utils/format'

export interface ServerInfo {
  uploadAddress: string
  downloadAddress: string
  presignEnabled: boolean
}

export interface FileEntry {
  name: string
  size: number
  isDir: boolean
  /** presign 开启时为预签名直链，否则为代理下载地址 */
  href?: string
}

export interface ListResult {
  path: string
  entries: FileEntry[]
  nextCursor: string
}

export interface SearchResult {
  path: string
  size: number
  href?: string
}

export interface SearchResponse {
  path: string
  results: SearchResult[]
  truncated: boolean
}

export interface CodeShow {
  language: string
  code: string
  downloadUrl: string
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function getJson<T>(url: string): Promise<T> {
  const resp = await fetch(url)
  if (!resp.ok) {
    let msg = `请求失败 (${resp.status})`
    try {
      const body = await resp.json()
      if (body && typeof body.error === 'string') msg = body.error
    } catch {
      /* 非 JSON 响应，保持默认文案 */
    }
    throw new ApiError(resp.status, msg)
  }
  return resp.json() as Promise<T>
}

export const api = {
  getInfo: () => getJson<ServerInfo>('/api/info'),

  list: (path: string, cursor = '', size = 100) =>
    getJson<ListResult>(`/api/files?path=${encodeURIComponent(path)}&cursor=${encodeURIComponent(cursor)}&size=${size}`),

  search: (path: string, q: string, limit = 200) =>
    getJson<SearchResponse>(`/api/files/search?path=${encodeURIComponent(path)}&q=${encodeURIComponent(q)}&limit=${limit}`),

  uploadCode: async (code: string, language: string): Promise<{ url: string }> => {
    const resp = await fetch('/api/code', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code, language }),
    })
    if (!resp.ok) {
      const body = await resp.json().catch(() => null)
      throw new ApiError(resp.status, body?.error ?? `上传失败 (${resp.status})`)
    }
    return resp.json()
  },

  getCode: (lang: string, hash: string) => getJson<CodeShow>(`/api/code/${encodeURIComponent(lang)}/${encodeURIComponent(hash)}`),
}

/** 删除文件（相对存储根的完整路径） */
export async function deleteFile(path: string): Promise<void> {
  const resp = await fetch(`/delete/${encodePath(path)}`, { method: 'DELETE' })
  if (!resp.ok) throw new ApiError(resp.status, `删除失败 (${resp.status})`)
}
