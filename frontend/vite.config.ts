import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const rootDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig(({ mode }) => {
  // Корень проекта — там лежат .env / .env.example с глобальными константами
  const env = loadEnv(mode, path.resolve(rootDir, '..'), '')
  const frontendPort = Number(env.FRONTEND_PORT || 5173)
  const apiProxyTarget = env.VITE_API_PROXY_TARGET || 'http://localhost:8080'
  const apiBasePath = env.VITE_API_BASE_PATH || '/v1'

  return {
    envDir: path.resolve(rootDir, '..'),
    plugins: [react(), tailwindcss()],
    server: {
      port: frontendPort,
      proxy: {
        [apiBasePath]: apiProxyTarget,
      },
    },
  }
})
