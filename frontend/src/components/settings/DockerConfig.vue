<script lang="ts" setup>
import { NForm, NFormItem, NInput, NInputNumber, NSwitch } from 'naive-ui'
import { useSettingsStore } from '@/stores/settingsStore'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
</script>

<template>
  <div class="config-form">
    <NForm
      label-placement="top"
      size="small"
      class="stacked-form"
    >
      <NFormItem :label="t('settings.docker.image')">
        <NInput
          v-model:value="settingsStore.settings.docker.image"
          :placeholder="t('settings.docker.imagePlaceholder')"
          class="mono-input"
        />
      </NFormItem>

      <div class="u-form-grid-2col">
        <NFormItem :label="t('settings.docker.memory')">
          <NInputNumber
            v-model:value="settingsStore.settings.docker.memoryLimit"
            :min="256"
            :max="32768"
            :step="256"
            class="u-w-full"
          />
        </NFormItem>

        <NFormItem :label="t('settings.docker.cpuLimit')">
          <NInputNumber
            v-model:value="settingsStore.settings.docker.cpuLimit"
            :min="1"
            :max="16"
            :step="1"
            class="u-w-full"
          />
        </NFormItem>
      </div>

      <NFormItem :label="t('settings.docker.workDir')">
        <NInput
          v-model:value="settingsStore.settings.docker.workDir"
          :placeholder="t('settings.docker.workDirPlaceholder')"
          class="mono-input"
        />
      </NFormItem>

      <NFormItem :label="t('settings.docker.network')">
        <NSwitch v-model:value="settingsStore.settings.docker.network">
          <template #checked>{{ t('common.enabled') }}</template>
          <template #unchecked>{{ t('common.disabled') }}</template>
        </NSwitch>
      </NFormItem>
    </NForm>

    <div class="info-box">
      <p class="info-text">
        {{ t('settings.docker.infoText') }}
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
