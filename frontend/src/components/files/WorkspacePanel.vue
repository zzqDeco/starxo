<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NButton, NEmpty, NIcon, NInput, NSpin, NTooltip, NTree, type TreeOption } from 'naive-ui'
import { CloudDownload, CloudUpload, CopyOutline, Refresh, Search, TrashOutline } from '@vicons/ionicons5'
import type { FileInfo, WorkspaceInfo } from '@/types/config'
import SplitHandle from '@/components/layout/SplitHandle.vue'
import FileTransfer from './FileTransfer.vue'
import CodePreview from './CodePreview.vue'
import { CleanupSandboxTmp, DownloadFile, GetWorkspaceInfo, ListWorkspaceFiles, ReadFilePreview } from '../../../wailsjs/go/service/FileService'
import { useI18n } from 'vue-i18n'
import { consumePendingWorkspacePath, onWorkspaceOpenPath } from '@/composables/useWorkspaceBridge'
import { useUiFeedback } from '@/composables/useUiFeedback'

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
const feedback = useUiFeedback()
const workspaceInfo = ref<WorkspaceInfo | null>(null)
const cleaningTmp = ref(false)

const selectedFile = computed(() => files.value.find(f => f.path === selectedPath.value) || null)
const workspacePath = computed(() => workspaceInfo.value?.workspacePath || '')
const workspaceHost = computed(() => {
  const info = workspaceInfo.value
  if (!info?.sshHost) return '-'
  return `${info.sshHost}:${info.sshPort || 22}`
})

const filteredFiles = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return files.value
  return files.value.filter(f => f.path.toLowerCase().includes(q) || displayPath(f.path).toLowerCase().includes(q) || f.name.toLowerCase().includes(q))
})

function displayPath(filePath: string) {
  const root = workspacePath.value.replace(/\/+$/, '')
  if (root && (filePath === root || filePath.startsWith(`${root}/`))) {
    return filePath.slice(root.length).replace(/^\/+/, '') || '.'
  }
  return filePath.replace(/^\/+/, '')
}

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
    const parts = displayPath(file.path).replace(/\\/g, '/').split('/').filter(Boolean)
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
    await refreshWorkspaceInfo()
    if (!workspaceInfo.value?.active) {
      files.value = []
      selectedPath.value = ''
      previewContent.value = ''
      return
    }
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

async function refreshWorkspaceInfo() {
  try {
    workspaceInfo.value = await GetWorkspaceInfo() as WorkspaceInfo
  } catch (e) {
    console.warn('Failed to inspect workspace:', e)
    workspaceInfo.value = null
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

async function copyWorkspacePath() {
  if (!workspacePath.value) return
  await navigator.clipboard.writeText(workspacePath.value)
  feedback.success(t('workspace.pathCopied'))
}

async function cleanupTmp() {
  const confirmed = await feedback.confirmDanger(t('workspace.cleanupTmpConfirm'))
  if (!confirmed) return
  cleaningTmp.value = true
  try {
    const result = await CleanupSandboxTmp()
    feedback.success(t('workspace.cleanupTmpDone', {
      count: result?.removedEntries || 0,
      size: formatBytes(result?.reclaimedBytes || 0),
    }))
    await refreshFiles()
  } catch (e) {
    feedback.error(t('workspace.cleanupTmp'), e)
  } finally {
    cleaningTmp.value = false
  }
}

function formatBytes(bytes: number) {
  if (!bytes || bytes < 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
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
        <NTooltip trigger="hover">
          <template #trigger>
            <NButton size="tiny" quaternary :disabled="!workspacePath" @click="copyWorkspacePath">
              <template #icon><NIcon size="14"><CopyOutline /></NIcon></template>
            </NButton>
          </template>
          {{ t('workspace.copyPath') }}
        </NTooltip>
        <NTooltip trigger="hover">
          <template #trigger>
            <NButton size="tiny" quaternary :disabled="!workspaceInfo?.active" :loading="cleaningTmp" @click="cleanupTmp">
              <template #icon><NIcon size="14"><TrashOutline /></NIcon></template>
            </NButton>
          </template>
          {{ t('workspace.cleanupTmp') }}
        </NTooltip>
      </div>
    </header>

    <div class="workspace-meta">
      <div class="meta-item">
        <span>{{ t('workspace.sandbox') }}</span>
        <strong>{{ workspaceInfo?.sandboxName || t('runtime.noActiveContainer') }}</strong>
      </div>
      <div class="meta-item">
        <span>{{ t('workspace.runtime') }}</span>
        <strong>{{ workspaceInfo?.runtime || '-' }}</strong>
      </div>
      <div class="meta-item">
        <span>{{ t('workspace.host') }}</span>
        <strong>{{ workspaceHost }}</strong>
      </div>
      <div class="meta-item path">
        <span>{{ t('workspace.path') }}</span>
        <strong>{{ workspacePath || '-' }}</strong>
      </div>
      <div class="meta-item">
        <span>{{ t('workspace.files') }}</span>
        <strong>{{ workspaceInfo?.fileCount || files.length }}</strong>
      </div>
      <div class="meta-item">
        <span>{{ t('workspace.size') }}</span>
        <strong>{{ formatBytes(workspaceInfo?.totalSize || 0) }}</strong>
      </div>
    </div>

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
            <NEmpty
              v-else
              size="small"
              :description="workspaceInfo?.active ? t('workspace.noFiles') : t('workspace.noActiveWorkspace')"
              class="tree-empty"
            />
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

.workspace-meta {
  display: grid;
  grid-template-columns: minmax(120px, 1.3fr) repeat(2, minmax(70px, 0.8fr)) minmax(160px, 2fr) repeat(2, minmax(60px, 0.6fr));
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-deepest);
  flex-shrink: 0;
}

.meta-item {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.meta-item span {
  color: var(--text-faint);
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.meta-item strong {
  min-width: 0;
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-item.path strong {
  font-family: var(--font-mono);
  font-weight: 500;
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

@media (max-width: 900px) {
  .workspace-meta {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
