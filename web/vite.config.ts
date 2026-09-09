import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const proxyTarget = process.env.VITE_API_PROXY || 'http://localhost:8080'

// 本地联调：Go 服务跑在 8080，前端 dev server 代理所有后端路由
const backendPaths = ['/api', '/upload', '/download', '/delete', '/archive', '/script']

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: Object.fromEntries(backendPaths.map((p) => [p, { target: proxyTarget, changeOrigin: true }])),
  },
})
