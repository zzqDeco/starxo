import { ref, computed, onMounted, onUnmounted } from 'vue'
import MarkdownIt from 'markdown-it'
import { escapeHtml, getHighlighter } from '@/composables/highlight'

let mdInstance: MarkdownIt | null = null

function getMd(): MarkdownIt {
  if (!mdInstance) {
    const hljs = getHighlighter()
    mdInstance = new MarkdownIt({
      html: false,
      linkify: true,
      typographer: true,
      highlight(str: string, lang: string) {
        const langLabel = lang || 'code'
        const headerHtml = `<div class="code-block-header"><span class="code-lang-label">${langLabel}</span><button class="code-copy-btn" onclick="navigator.clipboard.writeText(this.closest('.hljs-code-block').querySelector('code').textContent)">Copy</button></div>`

        if (lang && hljs.getLanguage(lang)) {
          try {
            return `<pre class="hljs-code-block">${headerHtml}<code class="hljs language-${lang}">${
              hljs.highlight(str, { language: lang, ignoreIllegals: true }).value
            }</code></pre>`
          } catch (_) { /* fallback */ }
        }
        return `<pre class="hljs-code-block">${headerHtml}<code class="hljs">${
          escapeHtml(str)
        }</code></pre>`
      }
    })
  }
  return mdInstance
}

export function useMarkdown() {
  function renderMarkdown(content: string): string {
    if (!content) return ''
    return getMd().render(content)
  }

  return { renderMarkdown }
}

export function useAutoScroll(containerRef: { value: HTMLElement | null }) {
  const isAutoScroll = ref(true)
  const isNearBottom = ref(true)

  function checkScroll() {
    const el = containerRef.value
    if (!el) return
    const threshold = 80
    isNearBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < threshold
  }

  function scrollToBottom(smooth = true) {
    const el = containerRef.value
    if (!el) return
    el.scrollTo({
      top: el.scrollHeight,
      behavior: smooth ? 'smooth' : 'instant'
    })
  }

  function onScroll() {
    checkScroll()
    isAutoScroll.value = isNearBottom.value
  }

  return { isAutoScroll, isNearBottom, scrollToBottom, onScroll }
}
