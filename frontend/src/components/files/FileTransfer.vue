<script lang="ts" setup>
import { ref } from 'vue'
import { NModal, NCard, NButton, NIcon, NProgress } from 'naive-ui'
import { CloudUpload } from '@vicons/ionicons5'
import { SelectAndUploadFile } from '../../../wailsjs/go/service/FileService'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  mode: 'upload' | 'download'
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
}>()

const uploading = ref(false)
const uploadResult = ref<string>('')

async function handleNativeUpload() {
  uploading.value = true
  uploadResult.value = ''
  try {
    const fileInfo = await SelectAndUploadFile()
    if (fileInfo) {
      uploadResult.value = `Uploaded: ${fileInfo.name}`
      setTimeout(() => {
        emit('update:show', false)
        uploading.value = false
        uploadResult.value = ''
      }, 800)
    } else {
      uploading.value = false
    }
  } catch (e: any) {
    uploadResult.value = `Upload failed: ${e?.message || e}`
    console.error('Upload failed:', e)
    uploading.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    @update:show="emit('update:show', $event)"
    preset="card"
    :title="mode === 'upload' ? t('files.uploadFileTitle') : t('files.downloadFileTitle')"
    :bordered="false"
    size="small"
    class="transfer-modal"
    :segmented="{ content: true }"
  >
    <div class="transfer-content">
      <!-- Upload -->
      <template v-if="mode === 'upload'">
        <div class="drop-zone" @click="handleNativeUpload">
          <div class="drop-content">
            <NIcon size="32" class="drop-icon">
              <CloudUpload />
            </NIcon>
            <p class="drop-text">{{ uploading ? t('files.uploading') : t('files.clickToSelect') }}</p>
            <p class="drop-hint">{{ t('files.uploadHint') }}</p>
            <p v-if="uploadResult" class="upload-result">{{ uploadResult }}</p>
          </div>
        </div>
      </template>
    </div>
  </NModal>
</template>

<style scoped>
.transfer-content {
  padding: 8px 0;
}

.drop-zone {
  border-color: var(--border-subtle) !important;
  background: var(--bg-deepest) !important;
  border-radius: var(--radius-lg) !important;
  transition: border-color var(--transition-fast);
}

.drop-zone:hover {
  border-color: var(--accent-cyan-dim) !important;
}

.drop-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px 16px;
}

.drop-icon {
  color: var(--text-faint);
  margin-bottom: 10px;
}

.drop-text {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 600;
  margin: 0 0 4px 0;
}

.drop-hint {
  color: var(--text-faint);
  font-size: 11px;
  margin: 0;
}
</style>
