import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import css from 'highlight.js/lib/languages/css'
import go from 'highlight.js/lib/languages/go'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import markdown from 'highlight.js/lib/languages/markdown'
import python from 'highlight.js/lib/languages/python'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'

let registered = false

export function getHighlighter() {
  if (!registered) {
    hljs.registerLanguage('bash', bash)
    hljs.registerLanguage('css', css)
    hljs.registerLanguage('go', go)
    hljs.registerLanguage('javascript', javascript)
    hljs.registerLanguage('json', json)
    hljs.registerLanguage('markdown', markdown)
    hljs.registerLanguage('python', python)
    hljs.registerLanguage('typescript', typescript)
    hljs.registerLanguage('xml', xml)
    hljs.registerAliases(['sh', 'shell', 'zsh'], { languageName: 'bash' })
    hljs.registerAliases(['js', 'jsx'], { languageName: 'javascript' })
    hljs.registerAliases(['ts', 'tsx'], { languageName: 'typescript' })
    hljs.registerAliases(['html', 'vue'], { languageName: 'xml' })
    registered = true
  }
  return hljs
}

export function escapeHtml(raw: string): string {
  return raw
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}
