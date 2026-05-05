<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NAlert, NButton, NCollapse, NCollapseItem, NIcon, NTag, NTooltip } from 'naive-ui'
import { BuildOutline, CheckmarkCircle, ClipboardOutline, CloseCircle, InformationCircle, Warning } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settingsStore'
import { useUiFeedback } from '@/composables/useUiFeedback'
import { DiagnoseSandboxRuntime, InstallSandboxRuntime } from '../../../wailsjs/go/service/SettingsService'
import type { SandboxDiagnosticsResult, SandboxDiagnosticCheck, SandboxFixSuggestion } from '@/types/config'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const feedback = useUiFeedback()

const diagnostics = ref<SandboxDiagnosticsResult | null>(null)
const installMessage = ref('')
const checking = ref(false)
const installing = ref(false)

const statusCounts = computed(() => {
  const counts = { pass: 0, warn: 0, fail: 0, info: 0, skipped: 0 }
  for (const check of diagnostics.value?.checks || []) {
    if (check.status in counts) {
      counts[check.status as keyof typeof counts] += 1
    }
  }
  return counts
})

function tagType(status: string) {
  if (status === 'pass') return 'success'
  if (status === 'fail') return 'error'
  if (status === 'warn') return 'warning'
  if (status === 'skipped') return 'default'
  return 'info'
}

function statusIcon(status: string) {
  if (status === 'pass') return CheckmarkCircle
  if (status === 'fail') return CloseCircle
  if (status === 'warn') return Warning
  return InformationCircle
}

function fixFor(check: SandboxDiagnosticCheck) {
  if (!check.fixIDs?.length || !diagnostics.value?.fixes?.length) return []
  const ids = new Set(check.fixIDs)
  return diagnostics.value.fixes.filter((fix) => ids.has(fix.id))
}

function riskType(risk: string) {
  if (risk === 'security') return 'error'
  if (risk === 'sudo') return 'warning'
  return 'info'
}

async function runDiagnostics() {
  checking.value = true
  installMessage.value = ''
  try {
    diagnostics.value = await DiagnoseSandboxRuntime(settingsStore.settings as any) as SandboxDiagnosticsResult
  } catch (e) {
    feedback.error(t('settings.sandbox.diagnoseRuntime'), e)
  } finally {
    checking.value = false
  }
}

async function installRuntime() {
  installing.value = true
  installMessage.value = ''
  try {
    const result = await InstallSandboxRuntime(settingsStore.settings as any)
    installMessage.value = result?.message || ''
    await runDiagnostics()
  } catch (e) {
    feedback.error(t('settings.sandbox.installRuntime'), e)
  } finally {
    installing.value = false
  }
}

async function copyFix(fix: SandboxFixSuggestion) {
  const text = (fix.commands || []).join('\n')
  if (!text) return
  await navigator.clipboard.writeText(text)
  feedback.success(t('settings.sandbox.copiedFix'))
}
</script>

<template>
  <section class="diagnostics-panel">
    <header class="diagnostics-head">
      <div class="diagnostics-title">
        <NIcon size="15"><BuildOutline /></NIcon>
        <span>{{ t('settings.sandbox.diagnostics') }}</span>
      </div>
      <div class="diagnostics-actions">
        <NButton size="small" :loading="checking" @click="runDiagnostics">
          {{ t('settings.sandbox.diagnoseRuntime') }}
        </NButton>
        <NButton size="small" type="primary" :loading="installing" @click="installRuntime">
          {{ t('settings.sandbox.installRuntime') }}
        </NButton>
      </div>
    </header>

    <NAlert v-if="installMessage" type="info" class="diagnostics-alert">
      {{ installMessage }}
    </NAlert>

    <template v-if="diagnostics">
      <div class="diagnostics-summary" :class="{ failed: !diagnostics.available }">
        <div class="summary-main">
          <strong>{{ diagnostics.summary }}</strong>
          <span>{{ diagnostics.runtime }} · {{ diagnostics.os || 'unknown' }}</span>
        </div>
        <div class="summary-tags">
          <NTag size="small" round type="success">{{ statusCounts.pass }} pass</NTag>
          <NTag v-if="statusCounts.warn" size="small" round type="warning">{{ statusCounts.warn }} warn</NTag>
          <NTag v-if="statusCounts.fail" size="small" round type="error">{{ statusCounts.fail }} fail</NTag>
        </div>
      </div>

      <div class="diagnostics-meta">
        <div><span>{{ t('settings.sandbox.rootDir') }}</span><code>{{ diagnostics.workspaceRoot || '-' }}</code></div>
        <div><span>{{ t('settings.sandbox.network') }}</span><code>{{ diagnostics.networkEnabled ? t('common.enabled') : t('common.disabled') }}</code></div>
        <div><span>{{ t('settings.sandbox.timeout') }}</span><code>{{ diagnostics.commandTimeoutSec }}s</code></div>
        <div><span>{{ t('settings.sandbox.memory') }}</span><code>{{ diagnostics.memoryLimitMB }}MB</code></div>
      </div>

      <div class="check-list">
        <article v-for="check in diagnostics.checks" :key="check.id" class="check-row">
          <NIcon size="15" class="check-icon" :class="check.status">
            <component :is="statusIcon(check.status)" />
          </NIcon>
          <div class="check-content">
            <div class="check-line">
              <span class="check-label">{{ check.label }}</span>
              <NTag size="small" round :type="tagType(check.status)">{{ check.status }}</NTag>
            </div>
            <p>{{ check.message }}</p>
            <p v-if="check.details" class="muted">{{ check.details }}</p>
            <NCollapse v-if="check.command || check.output || fixFor(check).length" class="check-collapse">
              <NCollapseItem :title="t('settings.sandbox.details')" :name="check.id">
                <pre v-if="check.command" class="code-block">{{ check.command }}</pre>
                <pre v-if="check.output" class="code-block">{{ check.output }}</pre>
                <div v-for="fix in fixFor(check)" :key="fix.id" class="inline-fix">
                  <NTag size="small" :type="riskType(fix.risk)">{{ fix.risk }}</NTag>
                  <span>{{ fix.title }}</span>
                </div>
              </NCollapseItem>
            </NCollapse>
          </div>
        </article>
      </div>

      <div v-if="diagnostics.fixes?.length" class="fix-list">
        <h4>{{ t('settings.sandbox.fixGuide') }}</h4>
        <article v-for="fix in diagnostics.fixes" :key="fix.id" class="fix-row">
          <div class="fix-head">
            <div>
              <strong>{{ fix.title }}</strong>
              <p>{{ fix.description }}</p>
            </div>
            <NTag size="small" :type="riskType(fix.risk)">{{ fix.risk }}</NTag>
          </div>
          <pre v-if="fix.commands?.length" class="code-block">{{ fix.commands.join('\n') }}</pre>
          <div v-if="fix.commands?.length" class="fix-actions">
            <NTooltip trigger="hover">
              <template #trigger>
                <NButton size="tiny" quaternary @click="copyFix(fix)">
                  <template #icon><NIcon size="13"><ClipboardOutline /></NIcon></template>
                  {{ t('settings.sandbox.copyFix') }}
                </NButton>
              </template>
              {{ fix.copyOnly ? t('settings.sandbox.copyOnly') : t('settings.sandbox.canAutoInstall') }}
            </NTooltip>
          </div>
        </article>
      </div>
    </template>
  </section>
</template>

<style scoped>
.diagnostics-panel {
  margin-top: 14px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-deepest);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.diagnostics-head,
.diagnostics-summary,
.check-row,
.fix-row {
  padding: 10px 12px;
}

.diagnostics-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-bottom: 1px solid var(--border-subtle);
}

.diagnostics-title,
.diagnostics-actions,
.summary-tags,
.check-line,
.fix-actions,
.inline-fix {
  display: flex;
  align-items: center;
  gap: 8px;
}

.diagnostics-title {
  min-width: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--text-secondary);
}

.diagnostics-actions {
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.diagnostics-alert {
  margin: 10px 12px 0;
}

.diagnostics-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-bottom: 1px solid var(--border-subtle);
}

.diagnostics-summary.failed {
  background: rgba(248, 113, 113, 0.06);
}

.summary-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.summary-main strong {
  font-size: 12px;
  color: var(--text-primary);
}

.summary-main span,
.muted,
.fix-head p {
  margin: 0;
  font-size: 11px;
  color: var(--text-muted);
}

.diagnostics-meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-subtle);
}

.diagnostics-meta div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.diagnostics-meta span {
  font-size: 10px;
  color: var(--text-faint);
}

.diagnostics-meta code,
.code-block {
  font-family: var(--font-mono);
  font-size: 11px;
}

.diagnostics-meta code {
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.check-row {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 8px;
  border-bottom: 1px solid var(--border-subtle);
}

.check-icon.pass {
  color: var(--accent-emerald);
}

.check-icon.fail {
  color: var(--accent-rose);
}

.check-icon.warn {
  color: var(--accent-amber);
}

.check-icon.info,
.check-icon.skipped {
  color: var(--text-faint);
}

.check-content {
  min-width: 0;
}

.check-line {
  justify-content: space-between;
  gap: 8px;
}

.check-label {
  min-width: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
}

.check-content p {
  margin: 3px 0 0;
  font-size: 11px;
  color: var(--text-muted);
  line-height: 1.45;
}

.check-collapse {
  margin-top: 6px;
}

.code-block {
  margin: 6px 0 0;
  max-height: 130px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--text-secondary);
  background: rgba(2, 6, 23, 0.7);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 7px;
}

.inline-fix {
  margin-top: 6px;
  font-size: 11px;
  color: var(--text-secondary);
}

.fix-list {
  padding: 10px 12px 12px;
}

.fix-list h4 {
  margin: 0 0 8px;
  font-size: 11px;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.6px;
}

.fix-row {
  padding: 9px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
}

.fix-row + .fix-row {
  margin-top: 8px;
}

.fix-head {
  display: flex;
  justify-content: space-between;
  gap: 10px;
}

.fix-head strong {
  font-size: 12px;
  color: var(--text-primary);
}

.fix-actions {
  justify-content: flex-end;
  margin-top: 6px;
}
</style>
