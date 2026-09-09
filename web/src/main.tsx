import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import './index.css'
import { Layout } from './components/Layout'
import { HomeView } from './views/HomeView'
import { FilesView } from './views/FilesView'
import { CodeUploadView } from './views/CodeUploadView'
import { CodeShowView } from './views/CodeShowView'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<HomeView />} />
          <Route path="/download" element={<FilesView />} />
          <Route path="/download/*" element={<FilesView />} />
          <Route path="/code" element={<CodeUploadView />} />
          <Route path="/code/:lang/:hash" element={<CodeShowView />} />
          <Route path="*" element={<p className="text-center text-sm text-slate-500">页面不存在</p>} />
        </Routes>
      </Layout>
    </BrowserRouter>
  </StrictMode>,
)
