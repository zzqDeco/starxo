<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NAlert, NButton, NIcon } from 'naive-ui'
import { ClipboardOutline, Reload } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import type { SSHConfig } from '@/types/config'
import { MAC_LOCAL_NETWORK_RESET_COMMAND } from '@/utils/sshErrorHints'
import { CheckMacLocalNetworkAccess } from '../../../wailsjs/go/service/SettingsService'

interface MacLocalNetworkCheckResult {
  appDialOK?: boolean
  cliAttempted?: boolean
  cliReachable?: boolean
  likelyPermissionIssue?: boolean
  confirmedPermissionIssue?: boolean
  summary?: string
  resetCommand?: string
}

const props = defineProps<{
  ssh: Partial<SSHConfig>
  compact?: boolean
}>()

const { t } = useI18n()
const diagnosing = ref(false)
const copied = ref(false)
const diagnostic = ref<MacLocalNetworkCheckResult | null>(null)
const diagnosticError = ref('')

const resetCommand = computed(() => diagnostic.value?.resetCommand || MAC_LOCAL_NETWORK_RESET_COMMAND)
const diagnosticSummary = computed(() => {
  if (diagnostic.value?.appDialOK) return t('settings.ssh.localNetwork.appDialOK')
  if (diagnostic.value?.confirmedPermissionIssue) return t('settings.ssh.localNetwork.confirmedSummary')
  if (diagnostic.value?.likelyPermissionIssue) return t('settings.ssh.localNetwork.likelySummary')
  if (diagnostic.value?.summary) return diagnostic.value.summary
  return ''
})

async function runDiagnostic() {
  diagnosing.value = true
  diagnosticError.value = ''
  try {
    diagnostic.value = await CheckMacLocalNetworkAccess(props.ssh as any)
  } catch (e) {
    diagnosticError.value = e instanceof Error ? e.message : String(e)
  } finally {
    diagnosing.value = false
  }
}

async function copyResetCommand() {
  await navigator.clipboard.writeText(resetCommand.value)
  copied.value = true
  window.setTimeout(() => { copied.value = false }, 1600)
}
</script>

<template>
  <NAlert type="warning" class="mac-local-network-card" :show-icon="false">
    <div class="fix-content">
      <div class="fix-copy">
        <div class="fix-title">{{ t('settings.ssh.localNetwork.title') }}</div>
        <p>{{ t('settings.ssh.localNetwork.body') }}</p>
        <ol v-if="!compact" class="fix-steps">
          <li>{{ t('settings.ssh.localNetwork.stepSettings') }}</li>
          <li>{{ t('settings.ssh.localNetwork.stepEnable') }}</li>
          <li>{{ t('settings.ssh.localNetwork.stepRestart') }}</li>
        </ol>
        <p v-if="diagnosticSummary" class="diagnostic-summary">{{ diagnosticSummary }}</p>
        <p v-if="diagnosticError" class="diagnostic-error">{{ diagnosticError }}</p>
      </div>
      <div class="fix-actions">
        <NButton size="tiny" secondary :loading="diagnosing" @click="runDiagnostic">
          <template #icon><NIcon><Reload /></NIcon></template>
          {{ t('settings.ssh.localNetwork.diagnose') }}
        </NButton>
        <NButton size="tiny" secondary @click="copyResetCommand">
          <template #icon><NIcon><ClipboardOutline /></NIcon></template>
          {{ copied ? t('settings.ssh.localNetwork.copied') : t('settings.ssh.localNetwork.copyReset') }}
        </NButton>
      </div>
      <code v-if="!compact" class="reset-command">{{ resetCommand }}</code>
    </div>
  </NAlert>
</template>

<style scoped>
.mac-local-network-card {
  margin-top: 8px;
}

.fix-content {
  display: grid;
  gap: 8px;
}

.fix-title {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--text-primary);
}

.fix-copy p {
  margin: 4px 0 0;
  color: var(--text-secondary);
  line-height: 1.45;
}

.fix-steps {
  margin: 8px 0 0;
  padding-left: 18px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.diagnostic-summary {
  color: var(--accent-amber) !important;
}

.diagnostic-error {
  color: var(--accent-rose) !important;
}

.fix-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.reset-command {
  display: block;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--fs-2xs);
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
