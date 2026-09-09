// @vitest-environment happy-dom
import { describe, expect, it } from 'vitest'
import { buildScriptUrl, codeViewLink, encodePath, formatSize, parseBrowserPath, scriptCopyCommand } from './format'

describe('encodePath', () => {
  it('逐段转义并去掉多余斜杠', () => {
    expect(encodePath('2025/09/我的 文件.txt')).toBe('2025/09/%E6%88%91%E7%9A%84%20%E6%96%87%E4%BB%B6.txt')
    expect(encodePath('/a//b/')).toBe('a/b')
    expect(encodePath('')).toBe('')
  })
})

describe('formatSize', () => {
  it('格式化字节单位', () => {
    expect(formatSize(512)).toBe('512 B')
    expect(formatSize(2048)).toBe('2.0 KB')
    expect(formatSize(5 * 1024 * 1024)).toBe('5.0 MB')
    expect(formatSize(3 * 1024 ** 3)).toBe('3.0 GB')
  })
})

describe('parseBrowserPath', () => {
  it('解析 /download 前缀的浏览路径', () => {
    expect(parseBrowserPath('/download/2025/09')).toEqual({ path: '2025/09', segments: ['2025', '09'] })
    expect(parseBrowserPath('/download')).toEqual({ path: '', segments: [] })
    expect(parseBrowserPath('/download/code/go')).toEqual({ path: 'code/go', segments: ['code', 'go'] })
  })
})

describe('codeViewLink', () => {
  it('code 目录下的文件生成代码展示链接', () => {
    expect(codeViewLink('code/go', '20241111-abcdef12.go')).toBe('/code/go/20241111-abcdef12')
    expect(codeViewLink('code/python', 'x.py')).toBe('/code/python/x')
  })
  it('非 code 目录返回 null', () => {
    expect(codeViewLink('2025/09', 'a.txt')).toBeNull()
    expect(codeViewLink('code/unknownlang', 'a.xx')).toBeNull()
  })
})

describe('script url', () => {
  it('生成脚本地址与复制命令', () => {
    expect(buildScriptUrl('2025/09', 'bash')).toBe('/script/download?path=2025%2F09&os=bash')
    expect(buildScriptUrl('2025/09', 'powershell', true)).toBe('/script/download?path=2025%2F09&os=powershell&download=1')
  })

  it('复制命令包含完整 origin 地址', () => {
    expect(scriptCopyCommand('', 'bash')).toMatch(/^bash <\(curl -fsSL 'http:\/\/localhost:\d+\/script\/download\?path=&os=bash'\)$/)
    expect(scriptCopyCommand('', 'powershell')).toMatch(/^irm 'http:\/\/localhost:\d+\/script\/download\?path=&os=powershell' \| iex$/)
  })
})
