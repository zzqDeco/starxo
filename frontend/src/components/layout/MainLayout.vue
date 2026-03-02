<script lang="ts" setup>
import { ref, computed } from 'vue'
import { useWindowSize } from '@vueuse/core'
import Header from './Header.vue'
import Sidebar from './Sidebar.vue'
import SplitHandle from './SplitHandle.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import TerminalPanel from '@/components/terminal/TerminalPanel.vue'
import FileExplorer from '@/components/files/FileExplorer.vue'
import ContainerPanel from '@/components/containers/ContainerPanel.vue'
import SettingsPanel from '@/components/settings/SettingsPanel.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const showSettings = ref(false)
const rightPanelTab = ref<'terminal' | 'files' | 'containers'>('terminal')
const showRightPanel = ref(true)

// Resizable panel widths
const leftWidth = ref(240)
const rightWidth = ref(380)

// Window auto-adapt
const { width: windowWidth } = useWindowSize()
const effectiveLeftWidth = computed(() => {
  if (windowWidth.value < 900) {
    return Math.min(leftWidth.value, 180)
  }
  return leftWidth.value
})

function toggleSettings() {
  showSettings.value = !showSettings.value
}

function toggleRightPanel() {
  showRightPanel.value = !showRightPanel.value
}

// Tab indicator positioning
const tabIndex = computed(() => {
  if (rightPanelTab.value === 'terminal') return 0
  if (rightPanelTab.value === 'files') return 1
  return 2
})
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
      <!-- Header Bar -->
      <Header
        @toggle-settings="toggleSettings"
        @toggle-right-panel="toggleRightPanel"
        :right-panel-visible="showRightPanel"
      />

      <!-- Main Content Area -->
      <div class="content-area">
        <!-- Chat Area -->
        <div class="chat-area">
          <ChatPanel />
        </div>

        <!-- Right Splitter -->
        <SplitHandle
          v-if="showRightPanel"
          direction="horizontal"
          :default-size="380"
          :min-size="280"
          :max-size="600"
          :reverse="true"
          storage-key="starxo-right-panel-width"
          @update:size="(v: number) => rightWidth = v"
        />

        <!-- Right Panel -->
        <div
          v-if="showRightPanel"
          class="right-panel"
          :style="{ width: rightWidth + 'px' }"
        >
          <div class="right-panel-tabs">
            <button
              :class="['tab-btn', { active: rightPanelTab === 'terminal' }]"
              @click="rightPanelTab = 'terminal'"
            >
              {{ t('layout.terminal') }}
            </button>
            <button
              :class="['tab-btn', { active: rightPanelTab === 'files' }]"
              @click="rightPanelTab = 'files'"
            >
              {{ t('layout.files') }}
            </button>
            <button
              :class="['tab-btn', { active: rightPanelTab === 'containers' }]"
              @click="rightPanelTab = 'containers'"
            >
              {{ t('layout.containers') }}
            </button>
            <!-- Sliding indicator -->
            <div
              class="tab-indicator"
              :style="{
                transform: `translateX(${tabIndex * 100}%)`,
                width: 'calc(100% / 3)'
              }"
            />
          </div>
          <div class="right-panel-content">
            <TerminalPanel v-show="rightPanelTab === 'terminal'" />
            <FileExplorer v-show="rightPanelTab === 'files'" />
            <ContainerPanel v-show="rightPanelTab === 'containers'" />
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Settings Modal -->
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

.chat-area {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.right-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-surface);
  border-left: 1px solid var(--border-subtle);
  flex-shrink: 0;
  overflow: hidden;
}

.right-panel-tabs {
  display: flex;
  position: relative;
  border-bottom: 1px solid var(--border-subtle);
  padding: 0;
  flex-shrink: 0;
}

.tab-btn {
  flex: 1;
  padding: 10px 16px;
  background: none;
  border: none;
  color: var(--text-muted);
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
  letter-spacing: 0.3px;
  text-transform: uppercase;
  z-index: 1;
}

.tab-btn:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}

.tab-btn.active {
  color: var(--accent-cyan);
  background: rgba(34, 211, 238, 0.06);
}

.tab-indicator {
  position: absolute;
  bottom: 0;
  left: 0;
  height: 2px;
  background: var(--accent-cyan);
  border-radius: 1px 1px 0 0;
  transition: transform 250ms ease-out;
  pointer-events: none;
}

.right-panel-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
