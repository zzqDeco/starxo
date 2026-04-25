import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  build: {
    chunkSizeWarningLimit: 650,
    rollupOptions: {
      output: {
        manualChunks: {
          vue: ['vue', 'pinia', 'vue-i18n', '@vueuse/core'],
          naive: ['naive-ui', '@vicons/ionicons5'],
          markdown: ['markdown-it', 'highlight.js/lib/core']
        }
      }
    }
  }
})
