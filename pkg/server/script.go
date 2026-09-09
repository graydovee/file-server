package server

import (
	"fmt"
	"strings"
)

// buildBashScript 生成批量下载 bash 脚本，文件落盘到 <dir>/<相对路径>（dir 为空时直接按完整相对路径落盘）
func buildBashScript(dir string, files []scriptFile, truncated bool, note string) string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("# fileserver 批量下载脚本\n")
	b.WriteString(fmt.Sprintf("# 目标目录: %s\n", displayDir(dir)))
	b.WriteString(fmt.Sprintf("# 文件数量: %d%s\n", len(files), truncatedNote(truncated)))
	b.WriteString(fmt.Sprintf("# %s\n", note))
	b.WriteString(`
set -euo pipefail

if ! command -v curl >/dev/null 2>&1; then
  echo "错误: 未找到 curl，请先安装 curl" >&2
  exit 1
fi

fetch() {
  local url="$1" out="$2"
  echo "下载 ${out}"
  mkdir -p "$(dirname "${out}")"
  curl -fL --retry 3 --progress-bar -o "${out}" "${url}"
}

`)

	if len(files) == 0 {
		b.WriteString("echo \"目录为空，没有需要下载的文件\"\n")
		return b.String()
	}

	for _, f := range files {
		b.WriteString(fmt.Sprintf("fetch %s %s\n", bashQuote(f.url), bashQuote(f.target)))
	}
	b.WriteString(fmt.Sprintf("echo \"完成，共下载 %d 个文件\"\n", len(files)))
	return b.String()
}

// buildPowerShellScript 生成批量下载 PowerShell 脚本
func buildPowerShellScript(dir string, files []scriptFile, truncated bool, note string) string {
	var b strings.Builder
	b.WriteString("# fileserver 批量下载脚本 (PowerShell)\n")
	b.WriteString(fmt.Sprintf("# 目标目录: %s\n", displayDir(dir)))
	b.WriteString(fmt.Sprintf("# 文件数量: %d%s\n", len(files), truncatedNote(truncated)))
	b.WriteString(fmt.Sprintf("# %s\n", note))
	b.WriteString("$ErrorActionPreference = \"Stop\"\n\n")
	b.WriteString(`function Fetch([string]$Url, [string]$Out) {
  Write-Host "下载 $Out"
  $dir = Split-Path -Parent $Out
  if ($dir) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
  Invoke-WebRequest -Uri $Url -OutFile $Out
}

`)

	if len(files) == 0 {
		b.WriteString("Write-Host \"目录为空，没有需要下载的文件\"\n")
		return b.String()
	}

	for _, f := range files {
		b.WriteString(fmt.Sprintf("Fetch %s %s\n", psQuote(f.url), psQuote(f.target)))
	}
	b.WriteString(fmt.Sprintf("Write-Host \"完成，共下载 %d 个文件\"\n", len(files)))
	return b.String()
}

func bashQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func displayDir(dir string) string {
	if dir == "" {
		return "根目录"
	}
	return dir
}

func truncatedNote(truncated bool) string {
	if truncated {
		return "（超出上限，仅包含部分文件）"
	}
	return ""
}
