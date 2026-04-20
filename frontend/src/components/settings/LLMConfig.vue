<script lang="ts" setup>
import { ref, computed } from 'vue'
import { NForm, NFormItem, NInput, NSelect, NButton, NIcon } from 'naive-ui'
import { Add, Checkmark, CloseOutline, Flash } from '@vicons/ionicons5'
import { useSettingsStore } from '@/stores/settingsStore'
import { TestLLMConnection } from '../../../wailsjs/go/service/SettingsService'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const testing = ref(false)
const testResult = ref<'success' | 'error' | null>(null)
const newHeaderKey = ref('')
const newHeaderValue = ref('')

function addHeader() {
  if (!newHeaderKey.value) return
  if (!settingsStore.settings.llm.headers) {
    settingsStore.settings.llm.headers = {}
  }
  settingsStore.settings.llm.headers[newHeaderKey.value] = newHeaderValue.value
  newHeaderKey.value = ''
  newHeaderValue.value = ''
}

function updateHeader(key: string, value: string) {
  if (settingsStore.settings.llm.headers) {
    settingsStore.settings.llm.headers[key] = value
  }
}

function removeHeader(key: string) {
  if (settingsStore.settings.llm.headers) {
    delete settingsStore.settings.llm.headers[key]
  }
}

const providerOptions = computed(() => [
  { label: t('settings.llm.providerOpenAI'), value: 'openai' },
  { label: t('settings.llm.providerDeepSeek'), value: 'deepseek' },
  { label: t('settings.llm.providerArk'), value: 'ark' },
  { label: t('settings.llm.providerOllama'), value: 'ollama' },
])

const defaultURLs: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  deepseek: 'https://api.deepseek.com/v1',
  ark: 'https://ark.cn-beijing.volces.com/api/v3',
  ollama: 'http://localhost:11434/v1'
}

function onProviderChange(val: string) {
  settingsStore.settings.llm.type = val as any
  if (defaultURLs[val]) {
    settingsStore.settings.llm.baseURL = defaultURLs[val]
  }
}

async function testLLM() {
  testing.value = true
  testResult.value = null
  try {
    await TestLLMConnection(settingsStore.settings.llm as any)
    testResult.value = 'success'
  } catch (e) {
    testResult.value = 'error'
    console.error('LLM test failed:', e)
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <div class="config-form">
    <NForm
      label-placement="left"
      label-width="110"
      size="small"
    >
      <NFormItem :label="t('settings.llm.provider')">
        <NSelect
          :value="settingsStore.settings.llm.type"
          :options="providerOptions"
          @update:value="onProviderChange"
        />
      </NFormItem>

      <NFormItem :label="t('settings.llm.baseURL')">
        <NInput
          v-model:value="settingsStore.settings.llm.baseURL"
          :placeholder="t('settings.llm.baseURLPlaceholder')"
          class="mono-input"
        />
      </NFormItem>

      <NFormItem :label="t('settings.llm.apiKey')">
        <NInput
          v-model:value="settingsStore.settings.llm.apiKey"
          type="password"
          show-password-on="click"
          :placeholder="t('settings.llm.apiKeyPlaceholder')"
          class="mono-input"
        />
      </NFormItem>

      <NFormItem :label="t('settings.llm.model')">
        <NInput
          v-model:value="settingsStore.settings.llm.model"
          :placeholder="t('settings.llm.modelPlaceholder')"
          class="mono-input"
        />
      </NFormItem>

      <NFormItem :label="t('settings.llm.headers')">
        <div class="headers-list">
          <div v-for="(_, key) in (settingsStore.settings.llm.headers || {})" :key="key" class="header-row">
            <NInput :value="String(key)" :placeholder="t('settings.llm.headerName')" size="small" readonly class="mono-input header-key-input" />
            <NInput :value="settingsStore.settings.llm.headers![String(key)]" :placeholder="t('settings.llm.headerValue')" size="small" class="mono-input header-value-input" @update:value="(v: string) => updateHeader(String(key), v)" />
            <NButton quaternary circle size="tiny" @click="removeHeader(String(key))">
              <template #icon><NIcon size="14"><CloseOutline /></NIcon></template>
            </NButton>
          </div>
          <div class="header-row">
            <NInput v-model:value="newHeaderKey" :placeholder="t('settings.llm.headerName')" size="small" class="mono-input header-key-input" />
            <NInput v-model:value="newHeaderValue" :placeholder="t('settings.llm.headerValue')" size="small" class="mono-input header-value-input" />
            <NButton quaternary circle size="tiny" @click="addHeader" :disabled="!newHeaderKey">
              <template #icon><NIcon size="14"><Add /></NIcon></template>
            </NButton>
          </div>
        </div>
      </NFormItem>
    </NForm>

    <div class="form-actions">
      <NButton
        size="small"
        :loading="testing"
        @click="testLLM"
        :type="testResult === 'success' ? 'success' : testResult === 'error' ? 'error' : 'default'"
      >
        <template #icon>
          <NIcon v-if="testResult === 'success'"><Checkmark /></NIcon>
          <NIcon v-else><Flash /></NIcon>
        </template>
        {{ testResult === 'success' ? t('settings.llm.working') : testResult === 'error' ? t('settings.llm.failed') : t('settings.llm.testModel') }}
      </NButton>
    </div>
  </div>
</template>

<style scoped>
.config-form {
  padding: 12px 0;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
  border-top: 1px solid var(--border-subtle);
  margin-top: 8px;
}

.mono-input :deep(input) {
  font-family: var(--font-mono) !important;
  font-size: 12px !important;
}

.headers-list {
  width: 100%;
}

.header-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.header-key-input {
  width: 40%;
}

.header-value-input {
  width: 50%;
}
</style>
