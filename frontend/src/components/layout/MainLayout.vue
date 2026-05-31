<script lang="ts" setup>
import { ref, computed, watch, defineAsyncComponent, onMounted, onUnmounted } from 'vue'
import { useWindowSize } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import Header from './Header.vue'
import Sidebar from './Sidebar.vue'
import SplitHandle from './SplitHandle.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import WorkspacePanel from '@/components/files/WorkspacePanel.vue'
import TerminalPanel from '@/components/terminal/TerminalPanel.vue'
import SxIconButton from '@/components/ui/SxIconButton.vue'
import SxInspectorSegmented from '@/components/ui/SxInspectorSegmented.vue'
import { useKeybinds } from '@/composables/useKeybinds'
import { useSessionStore } from '@/stores/sessionStore'
import { useUiFeedback } from '@/composables/useUiFeedback'
import { onWorkspaceOpenPath } from '@/composables/useWorkspaceBridge'
import { toggleWindowZoom } from '@/composables/useNativeWindow'

const ContainerDock = defineAsyncComponent(() => import('@/components/containers/ContainerDock.vue'))
const SettingsPanel = defineAsyncComponent(() => import('@/components/settings/SettingsPanel.vue'))
const CommandPalette = defineAsyncComponent(() => import('@/components/palette/CommandPalette.vue'))
const RuntimeTasksPanel = defineAsyncComponent(() => import('@/components/runtime/RuntimeTasksPanel.vue'))

type InspectorMode = 'runtime' | 'workspace' | 'terminal' | 'tasks'

const { t } = useI18n()
const sessionStore = useSessionStore()
const feedback = useUiFeedback()

const showSettings = ref(false)
const showMobileSidebar = ref(false)
const showResponsiveDock = ref(false)
const showPalette = ref(false)
const inspectorMode = ref<InspectorMode>('runtime')

useKeybinds([
  { combo: { key: 'k', meta: true }, handler: () => { showPalette.value = !showPalette.value }, allowInInput: true },
  { combo: { key: 'n', meta: true }, handler: async () => {
      try { await sessionStore.createSession() }
      catch (e) { feedback.error(t('feedback.actions.createSession'), e) }
    } },
  { combo: { key: ',', meta: true }, handler: () => { showSettings.value = true } },
  ...Array.from({ length: 9 }, (_, i) => ({
    combo: { key: String(i + 1), meta: true },
    handler: async () => {
      const sess = sessionStore.sessions[i]
      if (!sess) return
      try { await sessionStore.switchSession(sess.id) }
      catch (e) { feedback.error(t('feedback.actions.switchSession'), e) }
    },
  })),
])

// Resizable panel widths
const leftWidth = ref(220)
const runtimeDockWidth = ref(344)
const workspaceInspectorWidth = ref(620)

// Window auto-adapt
const { width: windowWidth } = useWindowSize()

const isBelow1200 = computed(() => windowWidth.value < 1200)
const isBelow992 = computed(() => windowWidth.value < 992)
const isBelow768 = computed(() => windowWidth.value <= 768)
const isDesktopInspector = computed(() => !isBelow1200.value)
const workspaceInspectorActive = computed(() => isDesktopInspector.value && inspectorMode.value === 'workspace')
const runtimeInspectorActive = computed(() => isDesktopInspector.value && inspectorMode.value === 'runtime')
const terminalInspectorActive = computed(() => isDesktopInspector.value && inspectorMode.value === 'terminal')
const tasksInspectorActive = computed(() => isDesktopInspector.value && inspectorMode.value === 'tasks')
const inspectorVisible = computed(() => isDesktopInspector.value)
const workspaceVisible = computed(() => inspectorMode.value === 'workspace' && (isDesktopInspector.value || showResponsiveDock.value))
const runtimeTasksVisible = computed(() => inspectorMode.value === 'tasks' && (isDesktopInspector.value || showResponsiveDock.value))
const isSidebarCompact = computed(() => !isBelow768.value && (workspaceInspectorActive.value || windowWidth.value < 1320))

const leftMinSize = computed(() => {
  if (isBelow768.value) return 0
  if (isBelow992.value) return 160
  return 188
})

const leftMaxSize = computed(() => {
  if (isBelow992.value) return 280
  return 320
})

const dockMinSize = computed(() => {
  if (workspaceInspectorActive.value) return 420
  if (isBelow992.value) return 240
  if (isBelow1200.value) return 280
  return 320
})

const effectiveLeftWidth = computed(() => {
  if (isBelow768.value) {
    return 0
  }
  if (isSidebarCompact.value) {
    return 64
  }
  if (isBelow992.value) {
    return Math.min(leftWidth.value, 220)
  }
  return Math.min(leftWidth.value, 260)
})

const effectiveDockWidth = computed(() => {
  if (workspaceInspectorActive.value) {
    const maxWorkspace = Math.max(dockMinSize.value, Math.min(760, windowWidth.value - effectiveLeftWidth.value - 460))
    return Math.min(Math.max(workspaceInspectorWidth.value, dockMinSize.value), maxWorkspace)
  }
  if (isBelow1200.value) {
    return Math.min(runtimeDockWidth.value, 360)
  }
  return Math.min(runtimeDockWidth.value, 500)
})

const inspectorDefaultSize = computed(() => workspaceInspectorActive.value ? 620 : 360)
const inspectorMaxSize = computed(() => {
  if (workspaceInspectorActive.value) {
    return Math.min(760, Math.max(460, windowWidth.value - effectiveLeftWidth.value - 460))
  }
  return 500
})
const inspectorStorageKey = computed(() => workspaceInspectorActive.value ? 'starxo-workspace-inspector-width' : 'starxo-runtime-inspector-width')

function updateInspectorWidth(value: number) {
  if (workspaceInspectorActive.value) {
    workspaceInspectorWidth.value = value
    return
  }
  runtimeDockWidth.value = value
}

watch(isBelow1200, (below) => {
  if (!below) {
    showResponsiveDock.value = false
  }
})

watch(isBelow768, (below) => {
  if (!below) {
    showMobileSidebar.value = false
  }
})

function toggleSettings() {
  showSettings.value = !showSettings.value
}

function toggleWorkspaceDrawer() {
  if (isDesktopInspector.value) {
    inspectorMode.value = workspaceInspectorActive.value ? 'runtime' : 'workspace'
    showResponsiveDock.value = false
    return
  }
  inspectorMode.value = 'workspace'
  showResponsiveDock.value = !showResponsiveDock.value
}

function toggleRuntimeTasks() {
  if (isDesktopInspector.value) {
    inspectorMode.value = tasksInspectorActive.value ? 'runtime' : 'tasks'
    showResponsiveDock.value = false
    return
  }
  inspectorMode.value = 'tasks'
  showResponsiveDock.value = !showResponsiveDock.value
}

function toggleTerminalInspector() {
  if (isDesktopInspector.value) {
    inspectorMode.value = terminalInspectorActive.value ? 'runtime' : 'terminal'
    showResponsiveDock.value = false
    return
  }
  inspectorMode.value = 'terminal'
  showResponsiveDock.value = !showResponsiveDock.value
}

function openWorkspaceDrawer() {
  if (isDesktopInspector.value) {
    inspectorMode.value = 'workspace'
    showResponsiveDock.value = false
    return
  }
  inspectorMode.value = 'workspace'
  showResponsiveDock.value = true
}

function openRuntimeTasks() {
  inspectorMode.value = 'tasks'
  showResponsiveDock.value = !isDesktopInspector.value
}

function openTerminalInspector() {
  inspectorMode.value = 'terminal'
  showResponsiveDock.value = !isDesktopInspector.value
}

function toggleMobileSidebar() {
  showMobileSidebar.value = !showMobileSidebar.value
}

function toggleResponsiveDock() {
  showResponsiveDock.value = !showResponsiveDock.value
}

function openCommandPalette() {
  showPalette.value = true
}

function toggleNativeWindowZoom() {
  void toggleWindowZoom()
}

let stopWorkspaceBridge: (() => void) | null = null
onMounted(() => {
  stopWorkspaceBridge = onWorkspaceOpenPath(() => {
    openWorkspaceDrawer()
  })
})

onUnmounted(() => {
  stopWorkspaceBridge?.()
})
</script>

<template>
  <div
    class="main-layout"
    :class="{
      'sidebar-compact': isSidebarCompact,
      'workspace-active': workspaceInspectorActive,
      'runtime-active': runtimeInspectorActive,
      'terminal-active': terminalInspectorActive,
      'tasks-active': tasksInspectorActive,
      'inspector-active': inspectorVisible,
    }"
  >
    <div
      class="window-top-edge-hit-area"
      aria-hidden="true"
      @dblclick.stop.prevent="toggleNativeWindowZoom"
    ></div>

    <div
      v-if="!isBelow768"
      class="left-panel"
      :style="{ width: effectiveLeftWidth + 'px' }"
    >
      <Sidebar :compact="isSidebarCompact" />
    </div>

    <SplitHandle
      v-if="!isBelow768 && !isSidebarCompact"
      direction="horizontal"
      :default-size="240"
      :min-size="leftMinSize"
      :max-size="leftMaxSize"
      storage-key="starxo-left-panel-width"
      @update:size="(v: number) => leftWidth = v"
    />

    <div class="center-section">
      <Header
        @toggle-settings="toggleSettings"
        @toggle-workspace-drawer="toggleWorkspaceDrawer"
        @toggle-runtime-tasks="toggleRuntimeTasks"
        @toggle-terminal="toggleTerminalInspector"
        @open-command-palette="openCommandPalette"
        :workspace-drawer-visible="workspaceVisible"
        :runtime-tasks-visible="runtimeTasksVisible"
      />

      <div class="content-area">
        <div class="chat-shell">
          <div class="chat-area">
            <ChatPanel />
          </div>
        </div>

        <template v-if="inspectorVisible">
          <SplitHandle
            :key="inspectorMode || 'none'"
            direction="horizontal"
            :default-size="inspectorDefaultSize"
            :min-size="dockMinSize"
            :max-size="inspectorMaxSize"
            :reverse="true"
            :storage-key="inspectorStorageKey"
            @update:size="updateInspectorWidth"
          />

          <aside class="inspector-panel" :class="`mode-${inspectorMode}`" :style="{ width: effectiveDockWidth + 'px' }">
            <div class="inspector-switcher">
              <SxInspectorSegmented v-model:active="inspectorMode" />
            </div>
            <div class="inspector-body">
              <WorkspacePanel v-if="inspectorMode === 'workspace'" />
              <TerminalPanel v-else-if="inspectorMode === 'terminal'" />
              <RuntimeTasksPanel v-else-if="inspectorMode === 'tasks'" :show="true" embedded />
              <ContainerDock v-else />
            </div>
          </aside>
        </template>
      </div>
    </div>

    <div
      v-if="isBelow1200"
      class="responsive-dock"
      :class="{ open: showResponsiveDock }"
    >
      <button type="button" class="dock-backdrop" @click="toggleResponsiveDock" />
      <div class="dock-panel-wrap">
        <aside class="dock-panel" :style="{ width: effectiveDockWidth + 'px' }">
          <div class="inspector-switcher">
            <SxInspectorSegmented v-model:active="inspectorMode" />
          </div>
          <div class="inspector-body">
            <WorkspacePanel v-if="inspectorMode === 'workspace'" />
            <TerminalPanel v-else-if="inspectorMode === 'terminal'" />
            <RuntimeTasksPanel v-else-if="inspectorMode === 'tasks'" :show="true" embedded />
            <ContainerDock v-else />
          </div>
        </aside>
      </div>
    </div>

    <div
      v-if="isBelow768"
      class="mobile-sidebar"
      :class="{ open: showMobileSidebar }"
    >
      <button type="button" class="mobile-backdrop" @click="toggleMobileSidebar" />
      <aside class="mobile-sidebar-panel">
        <Sidebar />
      </aside>
    </div>

    <SxIconButton
      v-if="isBelow1200"
      class="dock-tab"
      icon="runtime"
      :label="showResponsiveDock ? t('header.hideContainers') : t('header.showContainers')"
      size="small"
      @click="toggleResponsiveDock"
    />

    <SxIconButton
      v-if="isBelow768"
      class="sidebar-tab"
      icon="session"
      :label="showMobileSidebar ? t('header.hideSessions') : t('header.showSessions')"
      size="small"
      @click="toggleMobileSidebar"
    />
  </div>

  <SettingsPanel v-model:show="showSettings" />
  <CommandPalette
    v-model:show="showPalette"
    @open-settings="showSettings = true"
    @open-workspace="openWorkspaceDrawer"
    @open-runtime-tasks="openRuntimeTasks"
    @open-terminal="openTerminalInspector"
  />
</template>

<style scoped>
.main-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  max-width: 100vw;
  background: var(--platform-bg-window);
  overflow: hidden;
  position: relative;
}

.window-top-edge-hit-area {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 12px;
  z-index: calc(var(--z-sticky, 20) + 3);
  --wails-draggable: no-drag;
}

:global(:root[data-platform="macos"] .window-top-edge-hit-area){
  display: none;
}

/* CSS safety net — keeps layout contained if JS breakpoints lag at resize.
   Structural mounting (v-if gates) stays in JS so drawer contents unmount
   when collapsed; these rules only clamp visuals. */
@media (max-width: 1200px) {
  .inspector-panel {
    display: none;
  }
}

@media (max-width: 768px) {
  .left-panel {
    display: none;
  }
}

.left-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--platform-bg-sidebar);
  backdrop-filter: blur(22px) saturate(1.18);
  border-right: 1px solid var(--border-subtle);
  flex-shrink: 0;
  overflow: hidden;
}

.sidebar-compact .left-panel {
  border-right-color: color-mix(in srgb, var(--border-subtle) 70%, transparent);
}

.center-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: transparent;
  position: relative;
}

.content-area {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
  padding: 0;
  gap: 0;
}

.chat-shell {
  flex: 1;
  min-width: 0;
  min-height: 0;
  position: relative;
  overflow: hidden;
  background: var(--platform-bg-content);
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.chat-area {
  height: 100%;
  overflow: hidden;
}

.chat-area :deep(.chat-panel) {
  --chat-content-max-width: min(960px, 100%);
}

.inspector-active .chat-area :deep(.chat-panel) {
  --chat-content-max-width: 100%;
  --chat-content-padding: 18px;
}

.inspector-panel {
  height: 100%;
  background: var(--platform-bg-elevated);
  backdrop-filter: blur(18px) saturate(1.08);
  border: 0;
  border-left: 1px solid var(--border-subtle);
  border-radius: 0;
  box-shadow: none;
  flex-shrink: 0;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.inspector-panel.mode-workspace {
  background: var(--platform-bg-elevated);
}

.inspector-switcher {
  flex-shrink: 0;
  padding: 10px 10px 8px;
  border-bottom: 1px solid var(--border-subtle);
  background: color-mix(in srgb, var(--platform-bg-toolbar) 82%, transparent);
}

.inspector-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

:global(:root[data-platform="macos"] .content-area){
  padding: 0;
}

:global(:root[data-platform="macos"] .chat-shell),
:global(:root[data-platform="macos"] .inspector-panel){
  border-radius: 0;
  border-top: 0;
  border-bottom: 0;
  box-shadow: none;
}

:global(:root[data-platform="macos"] .chat-shell){
  border-left: 0;
}

:global(:root[data-platform="macos"] .inspector-panel){
  border-right: 0;
}

.responsive-dock,
.mobile-sidebar {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 80;
}

.dock-backdrop,
.mobile-backdrop {
  position: absolute;
  inset: 0;
  border: none;
  background: rgba(0, 0, 0, 0.22);
  backdrop-filter: blur(4px);
  opacity: 0;
  transition: opacity 180ms ease;
}

.dock-panel-wrap {
  position: absolute;
  top: 48px;
  right: 0;
  bottom: 0;
  transform: translateX(100%);
  transition: transform 220ms ease;
}

.dock-panel {
  height: 100%;
  border-left: 1px solid var(--border-subtle);
  background: var(--platform-bg-elevated);
  backdrop-filter: blur(22px) saturate(1.2);
  box-shadow: -20px 0 36px rgba(0, 0, 0, 0.18);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.mobile-sidebar-panel {
  position: absolute;
  top: 48px;
  left: 0;
  bottom: 0;
  width: min(300px, 86vw);
  border-right: 1px solid var(--border-subtle);
  background: var(--platform-bg-sidebar);
  backdrop-filter: blur(22px) saturate(1.2);
  transform: translateX(-100%);
  transition: transform 220ms ease;
  box-shadow: 20px 0 36px rgba(0, 0, 0, 0.18);
}

.responsive-dock.open,
.mobile-sidebar.open {
  pointer-events: auto;
}

.responsive-dock.open .dock-backdrop,
.mobile-sidebar.open .mobile-backdrop {
  opacity: 1;
}

.responsive-dock.open .dock-panel-wrap {
  transform: translateX(0);
}

.mobile-sidebar.open .mobile-sidebar-panel {
  transform: translateX(0);
}

.dock-tab,
.sidebar-tab {
  position: absolute;
  z-index: 90;
  top: 56px;
}

.dock-tab {
  right: 10px;
}

.sidebar-tab {
  left: 10px;
}
</style>
