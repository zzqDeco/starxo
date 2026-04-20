<script lang="ts" setup>
import { ref } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NButton, NIcon, NSpace } from 'naive-ui'
import { Checkmark, Key } from '@vicons/ionicons5'
import { useSettingsStore } from '@/stores/settingsStore'
import { TestSSHConnection } from '../../../wailsjs/go/service/SettingsService'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const testing = ref(false)
const testResult = ref<'success' | 'error' | null>(null)

async function testConnection() {
  testing.value = true
  testResult.value = null
  try {
    await TestSSHConnection(settingsStore.settings.ssh as any)
    testResult.value = 'success'
  } catch (e) {
    testResult.value = 'error'
    console.error('SSH test failed:', e)
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
      <NFormItem :label="t('settings.ssh.host')">
        <NInput
          v-model:value="settingsStore.settings.ssh.host"
          :placeholder="t('settings.ssh.hostPlaceholder')"
        />
      </NFormItem>

      <NFormItem :label="t('settings.ssh.port')">
        <NInputNumber
          v-model:value="settingsStore.settings.ssh.port"
          :min="1"
          :max="65535"
          class="u-w-full"
        />
      </NFormItem>

      <NFormItem :label="t('settings.ssh.username')">
        <NInput
          v-model:value="settingsStore.settings.ssh.user"
          :placeholder="t('settings.ssh.userPlaceholder')"
        />
      </NFormItem>

      <NFormItem :label="t('settings.ssh.password')">
        <NInput
          v-model:value="settingsStore.settings.ssh.password"
          type="password"
          show-password-on="click"
          :placeholder="t('settings.ssh.passwordPlaceholder')"
        />
      </NFormItem>

      <NFormItem :label="t('settings.ssh.privateKey')">
        <NInput
          v-model:value="settingsStore.settings.ssh.privateKey"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 4 }"
          :placeholder="t('settings.ssh.privateKeyPlaceholder')"
          class="mono-input"
        />
      </NFormItem>
    </NForm>

    <div class="form-actions">
      <NButton
        size="small"
        :loading="testing"
        @click="testConnection"
        :type="testResult === 'success' ? 'success' : testResult === 'error' ? 'error' : 'default'"
      >
        <template #icon>
          <NIcon v-if="testResult === 'success'"><Checkmark /></NIcon>
          <NIcon v-else><Key /></NIcon>
        </template>
        {{ testResult === 'success' ? t('settings.ssh.connected') : testResult === 'error' ? t('settings.ssh.failed') : t('settings.ssh.testConnection') }}
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

.mono-input :deep(textarea) {
  font-family: var(--font-mono) !important;
  font-size: 11px !important;
}
</style>
