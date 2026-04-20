<script lang="ts" setup>
import { computed } from 'vue'
import { NButton, NIcon, NSpin } from 'naive-ui'
import { CopyOutline } from '@vicons/ionicons5'
import hljs from 'highlight.js'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  path?: string
  content?: string
  loading?: boolean
  fileSize?: number
}>(), {
  path: '',
  content: '',
  loading: false,
  fileSize: 0,
})
const { t } = useI18n()

const extension = computed(() => {
  const idx = props.path.lastIndexOf('.')
  return idx > -1 ? props.path.slice(idx + 1).toLowerCase() : ''
})

const breadcrumb = computed(() => {
  if (!props.path) return []
  return props.path.replace(/\\/g, '/').split('/').filter(Boolean)
})

function languageForExt(ext: string): string {
  const map: Record<string, string> = {
    ts: 'typescript',
    tsx: 'typescript',
    js: 'javascript',
    jsx: 'javascript',
    vue: 'xml',
    go: 'go',
    py: 'python',
    sh: 'bash',
    md: 'markdown',
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    html: 'xml',
    css: 'css',
    scss: 'scss',
    sql: 'sql',
    java: 'java',
    rs: 'rust',
    c: 'c',
    cpp: 'cpp',
    h: 'c',
  }
  return map[ext] || ''
}

function escapeHtml(raw: string): string {
  return raw
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

const highlightedLines = computed(() => {
  const raw = props.content || ''
  if (!raw) return [] as string[]

  const lang = languageForExt(extension.value)
  let highlighted = ''

  try {
    if (lang && hljs.getLanguage(lang)) {
      highlighted = hljs.highlight(raw, { language: lang, ignoreIllegals: true }).value
    } else {
      highlighted = escapeHtml(raw)
    }
  } catch {
    highlighted = escapeHtml(raw)
  }

  return highlighted.split(/\r?\n/)
})

const lineCount = computed(() => highlightedLines.value.length)

function copyAll() {
  if (!props.content) return
  navigator.clipboard.writeText(props.content)
}

function formatSize(bytes: number): string {
  if (!bytes || bytes <= 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  const idx = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)))
  return `${(bytes / Math.pow(1024, idx)).toFixed(idx > 0 ? 1 : 0)} ${units[idx]}`
}
</script>

<template>
  <section class="code-preview">
    <div class="code-header">
      <div class="code-meta">
        <div v-if="breadcrumb.length > 0" class="code-breadcrumb" :title="path">
          <template v-for="(seg, i) in breadcrumb" :key="i">
            <span class="crumb-sep" v-if="i > 0">/</span>
            <span :class="['crumb-seg', { leaf: i === breadcrumb.length - 1 }]">{{ seg }}</span>
          </template>
        </div>
        <div v-else class="code-breadcrumb code-breadcrumb-empty">
          {{ t('codePreview.noFileSelected') }}
        </div>
        <div class="code-stats">
          <span>{{ extension || 'text' }}</span>
          <span>·</span>
          <span>{{ formatSize(fileSize) }}</span>
          <span v-if="content">·</span>
          <span v-if="content">{{ t('codePreview.lines', { count: lineCount }) }}</span>
        </div>
      </div>
      <div class="code-actions">
        <NButton size="tiny" quaternary :disabled="!content" @click="copyAll">
          <template #icon><NIcon size="14"><CopyOutline /></NIcon></template>
          {{ t('codePreview.copy') }}
        </NButton>
      </div>
    </div>

    <NSpin :show="loading" size="small" class="code-body">
      <div v-if="!path" class="code-empty">{{ t('codePreview.noFileSelected') }}</div>

      <div v-else-if="!content" class="code-empty">{{ t('codePreview.emptyOrUnavailable') }}</div>

      <div v-else class="code-scroll">
        <table class="code-table">
          <tbody>
            <tr v-for="(line, index) in highlightedLines" :key="index" class="code-row">
              <td class="line-number">{{ index + 1 }}</td>
              <td class="line-content"><span v-html="line || '&nbsp;'" /></td>
            </tr>
          </tbody>
        </table>
      </div>
    </NSpin>
  </section>
</template>

<style scoped>
.code-preview {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--bg-deepest);
}

.code-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-bottom: 1px solid var(--border-subtle);
  padding: 8px 12px;
  background: var(--bg-elevated);
  flex-shrink: 0;
}

.code-meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.code-breadcrumb {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-muted);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  display: flex;
  align-items: center;
  gap: 2px;
}

.code-breadcrumb-empty {
  color: var(--text-faint);
  font-style: italic;
}

.crumb-sep {
  color: var(--text-faint);
  margin: 0 2px;
}

.crumb-seg {
  color: var(--text-muted);
}

.crumb-seg.leaf {
  color: var(--accent-cyan);
  font-weight: var(--fw-medium);
}

.code-stats {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  display: flex;
  gap: 5px;
}

.code-body {
  flex: 1;
  min-height: 0;
}

.code-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-faint);
  font-size: 12px;
}

.code-scroll {
  height: 100%;
  overflow: auto;
}

.code-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.code-row:hover {
  background: rgba(255, 255, 255, 0.02);
}

.line-number {
  width: 54px;
  text-align: right;
  vertical-align: top;
  padding: 0 10px;
  color: var(--text-faint);
  user-select: none;
  border-right: 1px solid rgba(255, 255, 255, 0.06);
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: 1.7;
  background: rgba(255, 255, 255, 0.01);
}

.line-content {
  vertical-align: top;
  padding: 0 12px;
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-secondary);
  white-space: pre;
}

.line-content :deep(.hljs-keyword),
.line-content :deep(.hljs-selector-tag) { color: #c792ea; }
.line-content :deep(.hljs-string),
.line-content :deep(.hljs-attr) { color: #c3e88d; }
.line-content :deep(.hljs-number),
.line-content :deep(.hljs-literal) { color: #f78c6c; }
.line-content :deep(.hljs-comment) { color: #636780; font-style: italic; }
.line-content :deep(.hljs-function .hljs-title),
.line-content :deep(.hljs-title.function_) { color: #82aaff; }
.line-content :deep(.hljs-type),
.line-content :deep(.hljs-built_in) { color: #ffcb6b; }
.line-content :deep(.hljs-variable) { color: #f07178; }
.line-content :deep(.hljs-params) { color: #d4d6e4; }
.line-content :deep(.hljs-property) { color: #82aaff; }
.line-content :deep(.hljs-meta) { color: #ff9cac; }
</style>
