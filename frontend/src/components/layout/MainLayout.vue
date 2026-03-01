<script lang="ts" setup>
import { ref } from 'vue'
import { NLayout, NLayoutSider, NLayoutContent } from 'naive-ui'
import Header from './Header.vue'
import Sidebar from './Sidebar.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import TerminalPanel from '@/components/terminal/TerminalPanel.vue'
import FileExplorer from '@/components/files/FileExplorer.vue'
import SettingsPanel from '@/components/settings/SettingsPanel.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const showSettings = ref(false)
const rightPanelTab = ref<'terminal' | 'files'>('terminal')
const showRightPanel = ref(true)

function toggleSettings() {
  showSettings.value = !showSettings.value
}

function toggleRightPanel() {
  showRightPanel.value = !showRightPanel.value
}
</script>

<template>
  <NLayout class="main-layout" has-sider position="absolute">
    <!-- Left Sidebar -->
    <NLayoutSider
      bordered
      :width="240"
      :native-scrollbar="false"
      content-style="display: flex; flex-direction: column; height: 100%;"
      class="left-sider"
    >
      <Sidebar />
    </NLayoutSider>

    <!-- Center Content -->
    <NLayout class="center-layout">
      <!-- Header Bar -->
      <Header
        @toggle-settings="toggleSettings"
        @toggle-right-panel="toggleRightPanel"
        :right-panel-visible="showRightPanel"
      />

      <!-- Main Content Area -->
      <NLayout has-sider class="content-area" position="absolute" style="top: 52px; bottom: 0;">
        <NLayoutContent class="chat-content">
          <ChatPanel />
        </NLayoutContent>

        <!-- Right Panel: Terminal + Files -->
        <NLayoutSider
          v-if="showRightPanel"
          bordered
          :width="380"
          :native-scrollbar="false"
          placement="right"
          class="right-sider"
          content-style="display: flex; flex-direction: column; height: 100%;"
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
          </div>
          <div class="right-panel-content">
            <TerminalPanel v-show="rightPanelTab === 'terminal'" />
            <FileExplorer v-show="rightPanelTab === 'files'" />
          </div>
        </NLayoutSider>
      </NLayout>
    </NLayout>
  </NLayout>

  <!-- Settings Modal -->
  <SettingsPanel v-model:show="showSettings" />
</template>

<style scoped>
.main-layout {
  height: 100vh;
  width: 100vw;
  background: var(--bg-base);
}

.left-sider {
  background: var(--bg-surface) !important;
  border-right: 1px solid var(--border-subtle) !important;
}

.center-layout {
  background: var(--bg-base);
  position: relative;
}

.content-area {
  background: var(--bg-base);
}

.chat-content {
  background: var(--bg-base);
}

.right-sider {
  background: var(--bg-surface) !important;
  border-left: 1px solid var(--border-subtle) !important;
}

.right-panel-tabs {
  display: flex;
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
}

.tab-btn:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}

.tab-btn.active {
  color: var(--accent-cyan);
}

.tab-btn.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 16px;
  right: 16px;
  height: 2px;
  background: var(--accent-cyan);
  border-radius: 1px 1px 0 0;
}

.right-panel-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
