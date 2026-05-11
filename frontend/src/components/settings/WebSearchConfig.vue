<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NButton, NForm, NFormItem, NIcon, NInput, NInputNumber, NSelect, NSwitch, NTag } from 'naive-ui'
import { Flash, Search } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settingsStore'
import { useUiFeedback } from '@/composables/useUiFeedback'
import { DiagnoseWebSearch } from '../../../wailsjs/go/service/SettingsService'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const feedback = useUiFeedback()
const diagnosing = ref(false)
const diagnostics = ref<any | null>(null)

const webSearch = computed(() => {
  if (!settingsStore.settings.agent.webSearch) {
    settingsStore.settings.agent.webSearch = { enabled: true, defaultProvider: 'duckduckgo', providers: [] }
  }
  if (!settingsStore.settings.agent.webSearch.providers) {
    settingsStore.settings.agent.webSearch.providers = []
  }
  return settingsStore.settings.agent.webSearch
})

const tinyfish = computed(() => webSearch.value.providers?.find((provider) => provider.name === 'tinyfish'))

const defaultProviderOptions = computed(() => {
  const configured = (webSearch.value.providers || [])
    .filter((provider) => provider.name)
    .map((provider) => ({ label: provider.name, value: provider.name }))
  return [
    { label: 'duckduckgo', value: 'duckduckgo' },
    { label: 'tinyfish', value: 'tinyfish' },
    ...configured.filter((provider) => provider.value !== 'tinyfish' && provider.value !== 'duckduckgo'),
  ]
})

function ensureTinyFish() {
  let provider = tinyfish.value
  if (!provider) {
    provider = { name: 'tinyfish', type: 'tinyfish' }
    webSearch.value.providers!.push(provider)
  }
  provider.type = 'tinyfish'
  provider.endpoint = provider.endpoint || 'https://api.search.tinyfish.ai'
  provider.apiKeyEnv = provider.apiKeyEnv || 'TINYFISH_API_KEY'
  provider.location = provider.location || 'US'
  provider.language = provider.language || 'en'
  webSearch.value.defaultProvider = 'tinyfish'
}

function removeTinyFish() {
  webSearch.value.providers = (webSearch.value.providers || []).filter((provider) => provider.name !== 'tinyfish')
  if (webSearch.value.defaultProvider === 'tinyfish') {
    webSearch.value.defaultProvider = 'duckduckgo'
  }
}

async function runDiagnostics() {
  diagnosing.value = true
  try {
    diagnostics.value = await DiagnoseWebSearch(settingsStore.settings as any)
  } catch (e) {
    feedback.error(t('settings.webSearch.diagnostics'), e)
  } finally {
    diagnosing.value = false
  }
}

function statusType(status?: string) {
  if (status === 'pass') return 'success'
  if (status === 'warn') return 'warning'
  if (status === 'fail') return 'error'
  return 'default'
}
</script>

<template>
  <div class="websearch-config">
    <NForm label-placement="top" size="small" class="stacked-form">
      <div class="u-form-grid-2col">
        <NFormItem :label="t('settings.webSearch.enabled')">
          <NSwitch v-model:value="webSearch.enabled" />
        </NFormItem>
        <NFormItem :label="t('settings.webSearch.defaultProvider')">
          <NSelect v-model:value="webSearch.defaultProvider" :options="defaultProviderOptions" />
        </NFormItem>
      </div>

      <section class="provider-section">
        <header class="section-head">
          <div>
            <strong>{{ t('settings.webSearch.tinyfish') }}</strong>
            <span>{{ tinyfish ? tinyfish.endpoint || 'https://api.search.tinyfish.ai' : t('settings.webSearch.notConfigured') }}</span>
          </div>
          <div class="section-actions">
            <NButton size="small" @click="ensureTinyFish">{{ t('settings.webSearch.useTinyFish') }}</NButton>
            <NButton v-if="tinyfish" size="small" quaternary @click="removeTinyFish">{{ t('common.remove') }}</NButton>
          </div>
        </header>

        <div v-if="tinyfish" class="provider-fields">
          <NFormItem :label="t('settings.webSearch.endpoint')">
            <NInput v-model:value="tinyfish.endpoint" class="mono-input" />
          </NFormItem>
          <div class="u-form-grid-2col">
            <NFormItem :label="t('settings.webSearch.apiKeyEnv')">
              <NInput v-model:value="tinyfish.apiKeyEnv" class="mono-input" />
            </NFormItem>
            <NFormItem :label="t('settings.webSearch.maxResults')">
              <NInputNumber v-model:value="tinyfish.maxResults" :min="1" :max="20" class="full-width" />
            </NFormItem>
          </div>
          <div class="u-form-grid-2col">
            <NFormItem :label="t('settings.webSearch.location')">
              <NInput v-model:value="tinyfish.location" class="mono-input" />
            </NFormItem>
            <NFormItem :label="t('settings.webSearch.language')">
              <NInput v-model:value="tinyfish.language" class="mono-input" />
            </NFormItem>
          </div>
        </div>
      </section>
    </NForm>

    <div class="form-actions">
      <NButton size="small" :loading="diagnosing" @click="runDiagnostics">
        <template #icon><NIcon><Flash /></NIcon></template>
        {{ t('settings.webSearch.diagnostics') }}
      </NButton>
    </div>

    <section v-if="diagnostics" class="diagnostics">
      <div class="diagnostics-head">
        <NIcon size="18"><Search /></NIcon>
        <strong>{{ diagnostics.summary }}</strong>
        <NTag size="small" :type="diagnostics.available ? 'success' : 'error'">
          {{ diagnostics.available ? t('settings.webSearch.ready') : t('settings.webSearch.needsFix') }}
        </NTag>
      </div>
      <div class="provider-results">
        <div v-for="provider in diagnostics.providers || []" :key="provider.name" class="provider-result">
          <span class="provider-name">{{ provider.name }}</span>
          <NTag size="small" :type="statusType(provider.status)">{{ provider.status }}</NTag>
          <span class="provider-message">{{ provider.message }}</span>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.websearch-config {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.provider-section {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 10px;
}

.section-head,
.diagnostics-head,
.provider-result {
  display: flex;
  align-items: center;
  gap: 8px;
}

.section-head {
  justify-content: space-between;
}

.section-head strong,
.provider-name {
  color: var(--text-primary);
}

.section-head span,
.provider-message {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.section-actions,
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}

.provider-fields {
  margin-top: 10px;
}

.mono-input :deep(input) {
  font-family: var(--font-mono) !important;
  font-size: 12px !important;
}

.full-width {
  width: 100%;
}

.diagnostics {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 1px solid var(--border-subtle);
  padding-top: 10px;
}

.provider-results {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.provider-result {
  min-height: 28px;
}

.provider-name {
  min-width: 90px;
  font-family: var(--font-mono);
  font-size: 12px;
}

.provider-message {
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>
