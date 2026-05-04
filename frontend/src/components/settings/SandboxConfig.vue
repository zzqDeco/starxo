<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInput, NInputNumber, NSelect, NSwitch } from 'naive-ui'
import { useSettingsStore } from '@/stores/settingsStore'
import { useI18n } from 'vue-i18n'
import { CheckSandboxRuntime, InstallSandboxRuntime } from '../../../wailsjs/go/service/SettingsService'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const checking = ref(false)
const installing = ref(false)
const runtimeResult = ref<any | null>(null)

const runtimeOptions = computed(() => [
  { label: t('settings.sandbox.runtimeAuto'), value: 'auto' },
  { label: t('settings.sandbox.runtimeBwrap'), value: 'bwrap' },
  { label: t('settings.sandbox.runtimeSeatbelt'), value: 'seatbelt' },
])

const packagesText = computed({
  get: () => settingsStore.settings.sandbox.pythonPackages.join(', '),
  set: (value: string) => {
    settingsStore.settings.sandbox.pythonPackages = value
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
  },
})

async function checkRuntime() {
  checking.value = true
  try {
    runtimeResult.value = await CheckSandboxRuntime(settingsStore.settings as any)
  } finally {
    checking.value = false
  }
}

async function installRuntime() {
  installing.value = true
  try {
    runtimeResult.value = await InstallSandboxRuntime(settingsStore.settings as any)
    await checkRuntime()
  } finally {
    installing.value = false
  }
}
</script>

<template>
  <div class="config-form">
    <NForm label-placement="top" size="small" class="stacked-form">
      <NFormItem :label="t('settings.sandbox.runtime')">
        <NSelect
          v-model:value="settingsStore.settings.sandbox.runtime"
          :options="runtimeOptions"
        />
      </NFormItem>

      <div class="u-form-grid-2col">
        <NFormItem :label="t('settings.sandbox.rootDir')">
          <NInput
            v-model:value="settingsStore.settings.sandbox.rootDir"
            :placeholder="t('settings.sandbox.rootDirPlaceholder')"
            class="mono-input"
          />
        </NFormItem>

        <NFormItem :label="t('settings.sandbox.workDirName')">
          <NInput
            v-model:value="settingsStore.settings.sandbox.workDirName"
            :placeholder="t('settings.sandbox.workDirNamePlaceholder')"
            class="mono-input"
          />
        </NFormItem>
      </div>

      <div class="u-form-grid-2col">
        <NFormItem :label="t('settings.sandbox.memory')">
          <NInputNumber
            v-model:value="settingsStore.settings.sandbox.memoryLimitMB"
            :min="128"
            :max="32768"
            :step="256"
            class="u-w-full"
          />
        </NFormItem>

        <NFormItem :label="t('settings.sandbox.timeout')">
          <NInputNumber
            v-model:value="settingsStore.settings.sandbox.commandTimeoutSec"
            :min="10"
            :max="3600"
            :step="10"
            class="u-w-full"
          />
        </NFormItem>
      </div>

      <NFormItem :label="t('settings.sandbox.pythonPackages')">
        <NInput
          v-model:value="packagesText"
          :placeholder="t('settings.sandbox.pythonPackagesPlaceholder')"
          class="mono-input"
        />
      </NFormItem>

      <div class="switch-row">
        <NFormItem :label="t('settings.sandbox.network')">
          <NSwitch v-model:value="settingsStore.settings.sandbox.network">
            <template #checked>{{ t('common.enabled') }}</template>
            <template #unchecked>{{ t('common.disabled') }}</template>
          </NSwitch>
        </NFormItem>

        <NFormItem :label="t('settings.sandbox.bootstrapPython')">
          <NSwitch v-model:value="settingsStore.settings.sandbox.bootstrapPython">
            <template #checked>{{ t('common.enabled') }}</template>
            <template #unchecked>{{ t('common.disabled') }}</template>
          </NSwitch>
        </NFormItem>
      </div>
    </NForm>

    <div class="runtime-actions">
      <NButton size="small" :loading="checking" @click="checkRuntime">
        {{ t('settings.sandbox.checkRuntime') }}
      </NButton>
      <NButton size="small" type="primary" :loading="installing" @click="installRuntime">
        {{ t('settings.sandbox.installRuntime') }}
      </NButton>
    </div>

    <NAlert v-if="runtimeResult" :type="runtimeResult.available || runtimeResult.installed ? 'success' : 'warning'" class="runtime-result">
      <div class="result-title">
        {{ runtimeResult.runtime || settingsStore.settings.sandbox.runtime }}
        <span v-if="runtimeResult.os">({{ runtimeResult.os }})</span>
      </div>
      <div>{{ runtimeResult.message }}</div>
      <div v-if="runtimeResult.missing?.length" class="result-muted">
        {{ t('settings.sandbox.missing') }}: {{ runtimeResult.missing.join(', ') }}
      </div>
    </NAlert>

    <div class="info-box">
      <p class="info-text">
        {{ t('settings.sandbox.infoText') }}
      </p>
    </div>
  </div>
</template>

<style scoped>
.config-form {
  padding: 12px 0;
}

.mono-input :deep(input),
.mono-input :deep(textarea) {
  font-family: var(--font-mono) !important;
  font-size: 12px !important;
}

.switch-row,
.runtime-actions {
  display: flex;
  gap: var(--space-md);
  align-items: center;
}

.runtime-actions {
  margin-top: 12px;
}

.runtime-result {
  margin-top: 12px;
}

.result-title {
  font-weight: var(--fw-semibold);
}

.result-muted {
  margin-top: 4px;
  color: var(--text-muted);
}

.info-box {
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.info-text {
  font-size: 11px;
  color: var(--text-muted);
  margin: 0;
  line-height: 1.5;
}
</style>
