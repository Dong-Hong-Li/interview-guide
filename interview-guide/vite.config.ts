import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  // 与仓库根目录 compose：宿主机 Go 为 8081；本机 go run 常用 8080，可在 .env.development 设 VITE_DEV_PROXY_TARGET
  const apiProxyTarget =
    env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:8081'

  return {
    plugins: [react()],
    build: {
      rollupOptions: {
        output: {
          // Vite 8 / Rolldown 要求 manualChunks 为函数，不能用静态对象映射
          manualChunks(id) {
            if (
              id.includes('node_modules/react/') ||
              id.includes('node_modules/react-dom') ||
              id.includes('node_modules/react-router-dom')
            ) {
              return 'react-vendor'
            }
            if (
              id.includes('node_modules/framer-motion') ||
              id.includes('node_modules/lucide-react')
            ) {
              return 'ui-vendor'
            }
            if (id.includes('node_modules/react-syntax-highlighter')) {
              return 'syntax-highlighter'
            }
          },
        },
      },
    },
    server: {
      host: '0.0.0.0',
      port: 5173,
      proxy: {
        '/api': {
          target: apiProxyTarget,
          changeOrigin: true,
        },
      },
    },
  }
})
