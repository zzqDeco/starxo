<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue'
import { NTree, NButton, NIcon, NEmpty, NTag, NSpin } from 'naive-ui'
import type { TreeOption } from 'naive-ui'
import { FolderOpen, Document, CloudUpload, CloudDownload, Refresh } from '@vicons/ionicons5'
import type { FileInfo } from '@/types/config'
import FileTransfer from './FileTransfer.vue'
import { ListWorkspaceFiles, DownloadFile, ReadFilePreview } from '../../../wailsjs/go/service/FileService'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const files = ref<FileInfo[]>([])
const loading = ref(false)
const selectedFile = ref<string | null>(null)
const showTransfer = ref(false)
const transferMode = ref<'upload' | 'download'>('upload')
const previewContent = ref('')
const previewLoading = ref(false)

const treeData = computed<TreeOption[]>(() => {
  const dirs: Record<string, TreeOption> = {}
  const root: TreeOption[] = []

  for (const file of files.value) {
    const parts = file.path.replace(/\\/g, '/').split('/')
    const fileName = parts[parts.length - 1]
    const dirPath = parts.slice(0, -1).join('/')

    const node: TreeOption = {
      key: file.path,
      label: fileName,
      isLeaf: true,
      prefix: () => null,
      suffix: () => null,
    }

    if (dirPath && dirPath !== '.') {
      if (!dirs[dirPath]) {
        dirs[dirPath] = {
          key: `dir:${dirPath}`,
          label: dirPath,
          children: [],
          isLeaf: false,
        }
        root.push(dirs[dirPath])
      }
      dirs[dirPath].children!.push(node)
    } else {
      root.push(node)
    }
  }

  return root
})

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

async function refreshFiles() {
  loading.value = true
  try {
    const result = await ListWorkspaceFiles()
    if (result) {
      files.value = result as unknown as FileInfo[]
    }
  } catch (e) {
    console.warn('Failed to list files:', e)
  } finally {
    loading.value = false
  }
}

async function handleDownload() {
  if (!selectedFile.value) return
  try {
    await DownloadFile(selectedFile.value)
  } catch (e) {
    console.error('Download failed:', e)
  }
}

function handleUploadClick() {
  transferMode.value = 'upload'
  showTransfer.value = true
}

function handleNodeSelect(keys: string[]) {
  selectedFile.value = keys[0] && !keys[0].startsWith('dir:') ? keys[0] : null
  previewContent.value = ''
  if (selectedFile.value) {
    loadPreview(selectedFile.value)
  }
}

async function loadPreview(path: string) {
  previewLoading.value = true
  try {
    const content = await ReadFilePreview(path)
    // Only update if the file is still selected
    if (selectedFile.value === path) {
      previewContent.value = content || ''
    }
  } catch (e) {
    previewContent.value = ''
  } finally {
    previewLoading.value = false
  }
}

onMounted(() => {
  refreshFiles()
})
</script>

<template>
  <div class="file-explorer">
    <div class="file-header">
      <span class="file-title">{{ t('files.workspace') }}</span>
      <div class="file-actions">
        <NButton
          quaternary
          circle
          size="tiny"
          @click="handleUploadClick"
          :title="t('files.uploadFile')"
        >
          <template #icon>
            <NIcon size="14"><CloudUpload /></NIcon>
          </template>
        </NButton>
        <NButton
          quaternary
          circle
          size="tiny"
          :disabled="!selectedFile"
          @click="handleDownload"
          :title="t('files.downloadSelected')"
        >
          <template #icon>
            <NIcon size="14"><CloudDownload /></NIcon>
          </template>
        </NButton>
        <NButton
          quaternary
          circle
          size="tiny"
          @click="refreshFiles"
          :loading="loading"
          :title="t('files.refresh')"
        >
          <template #icon>
            <NIcon size="14"><Refresh /></NIcon>
          </template>
        </NButton>
      </div>
    </div>

    <div class="file-content">
      <NSpin :show="loading" size="small">
        <NTree
          v-if="treeData.length > 0"
          :data="treeData"
          block-line
          selectable
          default-expand-all
          @update:selected-keys="handleNodeSelect"
          class="file-tree"
        />
        <NEmpty
          v-else
          :description="t('files.noFiles')"
          size="small"
          class="file-empty"
        />
      </NSpin>
    </div>

    <!-- Selected file info -->
    <div v-if="selectedFile" class="file-info-bar">
      <span class="file-info-name">{{ selectedFile.split('/').pop() }}</span>
      <span class="file-info-size">
        {{ formatSize(files.find(f => f.path === selectedFile)?.size || 0) }}
      </span>
    </div>

    <!-- File preview -->
    <div v-if="selectedFile" class="file-preview">
      <NSpin :show="previewLoading" size="small">
        <pre v-if="previewContent" class="preview-code">{{ previewContent }}</pre>
        <div v-else class="preview-empty">{{ t('files.noPreview') }}</div>
      </NSpin>
    </div>

    <!-- Transfer dialog -->
    <FileTransfer
      v-model:show="showTransfer"
      :mode="transferMode"
    />
  </div>
</template>

<style scoped>
.file-explorer {
  display: flex;
  flex-direction: column;
  height: 100%;
  flex: 1;
}

.file-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.file-title {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.8px;
}

.file-actions {
  display: flex;
  gap: 2px;
}

.file-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px;
}

.file-tree {
  font-size: 12px;
}

.file-empty {
  padding: 32px 16px;
}

.file-info-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
  flex-shrink: 0;
}

.file-info-name {
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-info-size {
  font-size: 10px;
  color: var(--text-muted);
  flex-shrink: 0;
}

.file-preview {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-deepest);
}

.preview-code {
  margin: 0;
  padding: 8px 12px;
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-all;
}

.preview-empty {
  padding: 16px;
  text-align: center;
  font-size: 11px;
  color: var(--text-faint);
}
</style>
