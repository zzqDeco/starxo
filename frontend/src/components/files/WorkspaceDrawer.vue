<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useWindowSize } from '@vueuse/core'
import { NButton, NIcon } from 'naive-ui'
import { Close } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import SplitHandle from '@/components/layout/SplitHandle.vue'
import WorkspacePanel from './WorkspacePanel.vue'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
}>()

const { t } = useI18n()
const { width: windowWidth } = useWindowSize()
const drawerWidth = ref(980)

const minDrawerWidth = computed(() => Math.floor(Math.min(560, windowWidth.value * 0.9)))
const maxDrawerWidth = computed(() => Math.max(900, Math.floor(windowWidth.value * 0.82)))
const effectiveDrawerWidth = computed(() => {
  const clamped = Math.min(drawerWidth.value, maxDrawerWidth.value)
  return Math.max(minDrawerWidth.value, clamped)
})

function closeDrawer() {
  emit('update:show', false)
}

function handleKeydown(e: KeyboardEvent) {
  if (!props.show) return
  if (e.key === 'Escape') {
    closeDrawer()
  }
}

watch(maxDrawerWidth, (max) => {
  if (drawerWidth.value > max) {
    drawerWidth.value = max
  }
})

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div class="workspace-drawer" :class="{ open: show }">
    <button type="button" class="workspace-backdrop" @click="closeDrawer" />

    <div class="workspace-group">
      <SplitHandle
        direction="horizontal"
        :default-size="980"
        :min-size="minDrawerWidth"
        :max-size="maxDrawerWidth"
        :reverse="true"
        storage-key="starxo-workspace-drawer-width"
        @update:size="(v: number) => drawerWidth = v"
      />

      <aside class="workspace-panel" :style="{ width: effectiveDrawerWidth + 'px' }">
        <header class="workspace-panel-head">
          <h3 class="workspace-panel-title">{{ t('workspace.drawerTitle') }}</h3>
          <NButton quaternary circle size="small" @click="closeDrawer">
            <template #icon>
              <NIcon size="16"><Close /></NIcon>
            </template>
          </NButton>
        </header>

        <div class="workspace-panel-body">
          <WorkspacePanel />
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.workspace-drawer {
  position: absolute;
  inset: 0;
  z-index: 80;
  pointer-events: none;
}

.workspace-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(5, 6, 16, 0.46);
  opacity: 0;
  transition: opacity 180ms ease;
  border: none;
  padding: 0;
}

.workspace-group {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  display: flex;
  transform: translateX(100%);
  transition: transform 220ms ease;
}

.workspace-panel {
  height: 100%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-surface);
  border-left: 1px solid var(--border-subtle);
  box-shadow: -24px 0 42px rgba(0, 0, 0, 0.35);
}

.workspace-panel-head {
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 10px 0 12px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
}

.workspace-panel-title {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.8px;
  text-transform: uppercase;
  color: var(--text-faint);
}

.workspace-panel-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.workspace-drawer.open {
  pointer-events: auto;
}

.workspace-drawer.open .workspace-backdrop {
  opacity: 1;
}

.workspace-drawer.open .workspace-group {
  transform: translateX(0);
}
</style>
