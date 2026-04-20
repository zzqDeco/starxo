<script lang="ts" setup>
import { ref, computed, watch } from 'vue'
import { useWindowSize } from '@vueuse/core'
import { NButton, NIcon, NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { Albums, ChatboxEllipses } from '@vicons/ionicons5'
import Header from './Header.vue'
import Sidebar from './Sidebar.vue'
import SplitHandle from './SplitHandle.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import WorkspaceDrawer from '@/components/files/WorkspaceDrawer.vue'
import ContainerDock from '@/components/containers/ContainerDock.vue'
import SettingsPanel from '@/components/settings/SettingsPanel.vue'

const { t } = useI18n()

const showSettings = ref(false)
const showWorkspaceDrawer = ref(false)
const showMobileSidebar = ref(false)
const showResponsiveDock = ref(false)

// Resizable panel widths
const leftWidth = ref(240)
const containerDockWidth = ref(360)

// Window auto-adapt
const { width: windowWidth } = useWindowSize()

const isBelow1200 = computed(() => windowWidth.value < 1200)
const isBelow992 = computed(() => windowWidth.value < 992)
const isBelow768 = computed(() => windowWidth.value < 768)

const leftMinSize = computed(() => {
  if (isBelow768.value) return 0
  if (isBelow992.value) return 160
  return 200
})

const leftMaxSize = computed(() => {
  if (isBelow992.value) return 280
  return 360
})

const dockMinSize = computed(() => {
  if (isBelow992.value) return 240
  if (isBelow1200.value) return 280
  return 320
})

const effectiveLeftWidth = computed(() => {
  if (isBelow768.value) {
    return 0
  }
  if (isBelow992.value) {
    return Math.min(leftWidth.value, 220)
  }
  return leftWidth.value
})

const effectiveDockWidth = computed(() => {
  if (isBelow1200.value) {
    return Math.min(containerDockWidth.value, 320)
  }
  return containerDockWidth.value
})

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
  showWorkspaceDrawer.value = !showWorkspaceDrawer.value
}

function toggleMobileSidebar() {
  showMobileSidebar.value = !showMobileSidebar.value
}

function toggleResponsiveDock() {
  showResponsiveDock.value = !showResponsiveDock.value
}
</script>

<template>
  <div class="main-layout">
    <div
      v-if="!isBelow768"
      class="left-panel"
      :style="{ width: effectiveLeftWidth + 'px' }"
    >
      <Sidebar />
    </div>

    <SplitHandle
      v-if="!isBelow768"
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
        :workspace-drawer-visible="showWorkspaceDrawer"
      />

      <div class="content-area">
        <div class="chat-shell">
          <div class="chat-area">
            <ChatPanel />
          </div>
          <WorkspaceDrawer v-model:show="showWorkspaceDrawer" />
        </div>

        <template v-if="!isBelow1200">
          <SplitHandle
            direction="horizontal"
            :default-size="360"
            :min-size="dockMinSize"
            :max-size="500"
            :reverse="true"
            storage-key="starxo-container-dock-width"
            @update:size="(v: number) => containerDockWidth = v"
          />

          <div class="container-dock" :style="{ width: effectiveDockWidth + 'px' }">
            <ContainerDock />
          </div>
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
          <ContainerDock />
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

    <NTooltip v-if="isBelow1200" trigger="hover" placement="left">
      <template #trigger>
        <NButton class="dock-tab" circle size="small" @click="toggleResponsiveDock">
          <template #icon>
            <NIcon size="16"><Albums /></NIcon>
          </template>
        </NButton>
      </template>
      {{ showResponsiveDock ? t('header.hideContainers') : t('header.showContainers') }}
    </NTooltip>

    <NTooltip v-if="isBelow768" trigger="hover" placement="right">
      <template #trigger>
        <NButton class="sidebar-tab" circle size="small" @click="toggleMobileSidebar">
          <template #icon>
            <NIcon size="16"><ChatboxEllipses /></NIcon>
          </template>
        </NButton>
      </template>
      {{ showMobileSidebar ? t('header.hideSessions') : t('header.showSessions') }}
    </NTooltip>
  </div>

  <SettingsPanel v-model:show="showSettings" />
</template>

<style scoped>
.main-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  max-width: 100vw;
  background: var(--bg-base);
  overflow: hidden;
  position: relative;
}

/* CSS safety net — keeps layout contained if JS breakpoints lag at resize.
   Structural mounting (v-if gates) stays in JS so drawer contents unmount
   when collapsed; these rules only clamp visuals. */
@media (max-width: 1200px) {
  .container-dock {
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
  background: var(--bg-surface);
  border-right: 1px solid var(--border-subtle);
  flex-shrink: 0;
  overflow: hidden;
}

.center-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--bg-base);
  position: relative;
}

.content-area {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}

.chat-shell {
  flex: 1;
  min-width: 0;
  min-height: 0;
  position: relative;
  overflow: hidden;
}

.chat-area {
  height: 100%;
  overflow: hidden;
}

.container-dock {
  height: 100%;
  background: var(--bg-surface);
  border-left: 1px solid var(--border-subtle);
  flex-shrink: 0;
  min-width: 0;
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
  background: rgba(5, 6, 16, 0.4);
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
  background: var(--bg-surface);
  box-shadow: -20px 0 28px rgba(0, 0, 0, 0.25);
}

.mobile-sidebar-panel {
  position: absolute;
  top: 48px;
  left: 0;
  bottom: 0;
  width: min(300px, 86vw);
  border-right: 1px solid var(--border-subtle);
  background: var(--bg-surface);
  transform: translateX(-100%);
  transition: transform 220ms ease;
  box-shadow: 20px 0 28px rgba(0, 0, 0, 0.25);
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
