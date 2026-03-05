<script lang="ts" setup>
import { ref, computed } from 'vue'
import { useWindowSize } from '@vueuse/core'
import Header from './Header.vue'
import Sidebar from './Sidebar.vue'
import SplitHandle from './SplitHandle.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import WorkspaceDrawer from '@/components/files/WorkspaceDrawer.vue'
import ContainerDock from '@/components/containers/ContainerDock.vue'
import SettingsPanel from '@/components/settings/SettingsPanel.vue'

const showSettings = ref(false)
const showWorkspaceDrawer = ref(false)

// Resizable panel widths
const leftWidth = ref(240)
const containerDockWidth = ref(360)

// Window auto-adapt
const { width: windowWidth } = useWindowSize()
const effectiveLeftWidth = computed(() => {
  if (windowWidth.value < 900) {
    return Math.min(leftWidth.value, 180)
  }
  return leftWidth.value
})
const effectiveDockWidth = computed(() => {
  if (windowWidth.value < 1280) {
    return Math.min(containerDockWidth.value, 320)
  }
  return containerDockWidth.value
})

function toggleSettings() {
  showSettings.value = !showSettings.value
}

function toggleWorkspaceDrawer() {
  showWorkspaceDrawer.value = !showWorkspaceDrawer.value
}
</script>

<template>
  <div class="main-layout">
    <!-- Left Sidebar -->
    <div class="left-panel" :style="{ width: effectiveLeftWidth + 'px' }">
      <Sidebar />
    </div>

    <!-- Left Splitter -->
    <SplitHandle
      direction="horizontal"
      :default-size="240"
      :min-size="180"
      :max-size="360"
      storage-key="starxo-left-panel-width"
      @update:size="(v: number) => leftWidth = v"
    />

    <!-- Center Section -->
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

        <SplitHandle
          direction="horizontal"
          :default-size="360"
          :min-size="300"
          :max-size="500"
          :reverse="true"
          storage-key="starxo-container-dock-width"
          @update:size="(v: number) => containerDockWidth = v"
        />

        <div class="container-dock" :style="{ width: effectiveDockWidth + 'px' }">
          <ContainerDock />
        </div>
      </div>
    </div>
  </div>

  <SettingsPanel v-model:show="showSettings" />
</template>

<style scoped>
.main-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  background: var(--bg-base);
  overflow: hidden;
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
</style>
