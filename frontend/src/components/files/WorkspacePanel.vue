<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NButton, NEmpty, NIcon, NInput, NSpin, NTree, type TreeOption } from 'naive-ui'
import { CloudDownload, CloudUpload, Refresh, Search } from '@vicons/ionicons5'
import type { FileInfo } from '@/types/config'
import SplitHandle from '@/components/layout/SplitHandle.vue'
import FileTransfer from './FileTransfer.vue'
import CodePreview from './CodePreview.vue'
import { DownloadFile, ListWorkspaceFiles, ReadFilePreview } from '../../../wailsjs/go/service/FileService'
import { useI18n } from 'vue-i18n'
import { consumePendingWorkspacePath, onWorkspaceOpenPath } from '@/composables/useWorkspaceBridge'

interface WorkspaceTreeNode extends TreeOption {
  key: string
  label: string
  path?: string
  isLeaf?: boolean
  children?: WorkspaceTreeNode[]
}

const files = ref<FileInfo[]>([])
const loading = ref(false)
const previewLoading = ref(false)
const selectedPath = ref('')
const previewContent = ref('')
const query = ref('')
const treeWidth = ref(220)
const showTransfer = ref(false)
const { t } = useI18n()

const selectedFile = computed(() => files.value.find(f => f.path === selectedPath.value) || null)

const filteredFiles = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return files.value
  return files.value.filter(f => f.path.toLowerCase().includes(q) || f.name.toLowerCase().includes(q))
})

function sortNodes(nodes: WorkspaceTreeNode[]) {
  nodes.sort((a, b) => {
    const aDir = !!a.children && a.children.length > 0
    const bDir = !!b.children && b.children.length > 0
    if (aDir !== bDir) return aDir ? -1 : 1
    return String(a.label).localeCompare(String(b.label))
  })
  for (const n of nodes) {
    if (n.children && n.children.length > 0) sortNodes(n.children)
  }
}

function buildTree(fileList: FileInfo[]): WorkspaceTreeNode[] {
  const root: WorkspaceTreeNode[] = []
  const index = new Map<string, WorkspaceTreeNode>()

  for (const file of fileList) {
    const parts = file.path.replace(/\\/g, '/').split('/').filter(Boolean)
    if (parts.length === 0) continue

    let currentChildren = root
    let currentPath = ''

    for (let i = 0; i < parts.length - 1; i++) {
      const part = parts[i]
      currentPath = currentPath ? `${currentPath}/${part}` : part
      const key = `dir:${currentPath}`
      let dirNode = index.get(key)

      if (!dirNode) {
        dirNode = {
          key,
          label: part,
          path: currentPath,
          isLeaf: false,
          children: [],
        }
        currentChildren.push(dirNode)
        index.set(key, dirNode)
      }

      currentChildren = dirNode.children || []
      dirNode.children = currentChildren
    }

    const leaf: WorkspaceTreeNode = {
      key: file.path,
      label: parts[parts.length - 1],
      path: file.path,
      isLeaf: true,
    }
    currentChildren.push(leaf)
  }

  sortNodes(root)
  return root
}

const treeData = computed<WorkspaceTreeNode[]>(() => buildTree(filteredFiles.value))

async function refreshFiles() {
  loading.value = true
  try {
    const result = await ListWorkspaceFiles()
    files.value = (result as unknown as FileInfo[]) || []

    if (selectedPath.value && !files.value.find(f => f.path === selectedPath.value)) {
      selectedPath.value = ''
      previewContent.value = ''
    }
  } catch (e) {
    console.warn('Failed to list files:', e)
  } finally {
    loading.value = false
  }
}

async function handleDownload() {
  if (!selectedPath.value) return
  try {
    await DownloadFile(selectedPath.value)
  } catch (e) {
    console.error('Download failed:', e)
  }
}

async function loadPreview(path: string) {
  previewLoading.value = true
  try {
    const content = await ReadFilePreview(path)
    if (selectedPath.value === path) {
      previewContent.value = content || ''
    }
  } catch {
    previewContent.value = ''
  } finally {
    previewLoading.value = false
  }
}

function handleSelect(keys: Array<string | number>) {
  const key = String(keys[0] || '')
  if (!key || key.startsWith('dir:')) return
  selectedPath.value = key
  previewContent.value = ''
  loadPreview(key)
}

function openUpload() {
  showTransfer.value = true
}

async function openPath(path: string) {
  if (!path) return
  if (files.value.length === 0) {
    await refreshFiles()
  }
  selectedPath.value = path
  previewContent.value = ''
  loadPreview(path)
}

let stopWorkspaceBridge: (() => void) | null = null

onMounted(async () => {
  await refreshFiles()
  const pending = consumePendingWorkspacePath()
  if (pending) {
    await openPath(pending)
  }
  stopWorkspaceBridge = onWorkspaceOpenPath((path) => {
    openPath(path)
  })
})

onUnmounted(() => {
  stopWorkspaceBridge?.()
})
</script>

<template>
  <section class="workspace-panel">
    <header class="workspace-header">
      <div class="workspace-title">{{ t('workspace.title') }}</div>
      <div class="workspace-actions">
        <NButton size="tiny" quaternary @click="openUpload">
          <template #icon><NIcon size="14"><CloudUpload /></NIcon></template>
          {{ t('workspace.upload') }}
        </NButton>
        <NButton size="tiny" quaternary :disabled="!selectedPath" @click="handleDownload">
          <template #icon><NIcon size="14"><CloudDownload /></NIcon></template>
          {{ t('workspace.download') }}
        </NButton>
        <NButton size="tiny" quaternary :loading="loading" @click="refreshFiles">
          <template #icon><NIcon size="14"><Refresh /></NIcon></template>
          {{ t('workspace.refresh') }}
        </NButton>
      </div>
    </header>

    <div class="workspace-body">
      <div class="tree-pane" :style="{ width: treeWidth + 'px' }">
        <div class="tree-toolbar">
          <NInput
            v-model:value="query"
            size="small"
            :placeholder="t('workspace.searchPlaceholder')"
            clearable
          >
            <template #prefix><NIcon size="14"><Search /></NIcon></template>
          </NInput>
        </div>

        <div class="tree-content">
          <NSpin :show="loading" size="small">
            <NTree
              v-if="treeData.length > 0"
              :data="treeData"
              block-line
              selectable
              default-expand-all
              :selected-keys="selectedPath ? [selectedPath] : []"
              @update:selected-keys="handleSelect"
              class="workspace-tree"
            />
            <NEmpty v-else size="small" :description="t('workspace.noFiles')" class="tree-empty" />
          </NSpin>
        </div>
      </div>

      <SplitHandle
        direction="horizontal"
        :default-size="220"
        :min-size="170"
        :max-size="420"
        storage-key="starxo-workspace-tree-width"
        @update:size="(v: number) => treeWidth = v"
      />

      <div class="preview-pane">
        <CodePreview
          :path="selectedPath"
          :content="previewContent"
          :loading="previewLoading"
          :file-size="selectedFile?.size || 0"
        />
      </div>
    </div>

    <FileTransfer v-model:show="showTransfer" mode="upload" />
  </section>
</template>

<style scoped>
.workspace-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--bg-surface);
}

.workspace-header {
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-bottom: 1px solid var(--border-subtle);
  padding: 0 12px;
  flex-shrink: 0;
}

.workspace-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.8px;
  color: var(--text-faint);
}

.workspace-actions {
  display: flex;
  gap: 4px;
}

.workspace-body {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
}

.tree-pane {
  display: flex;
  flex-direction: column;
  min-width: 0;
  border-right: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
}

.tree-toolbar {
  padding: 8px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.tree-content {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 4px;
}

.workspace-tree {
  font-size: 12px;
}

.workspace-tree :deep(.n-tree-node-content) {
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: background var(--transition-ui), color var(--transition-ui);
}

.workspace-tree :deep(.n-tree-node-content:hover) {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.workspace-tree :deep(.n-tree-node--selected .n-tree-node-content) {
  position: relative;
  background: var(--bg-elevated);
  color: var(--accent-cyan);
}

.workspace-tree :deep(.n-tree-node--selected .n-tree-node-content::before) {
  content: "";
  position: absolute;
  left: -2px;
  top: 4px;
  bottom: 4px;
  width: 2px;
  background: var(--accent-cyan);
  border-radius: 1px;
}

.tree-empty {
  padding: 20px 8px;
}

.preview-pane {
  flex: 1;
  min-width: 0;
  min-height: 0;
}
</style>
