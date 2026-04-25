<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NIcon } from 'naive-ui'
import { Cube, Terminal, RadioButtonOn } from '@vicons/ionicons5'
import ContainerPanel from './ContainerPanel.vue'
import TerminalPanel from '@/components/terminal/TerminalPanel.vue'
import { useContainerStore } from '@/stores/containerStore'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const containerStore = useContainerStore()
const activeTab = ref<'containers' | 'terminal'>('containers')

const activeContainer = computed(() =>
  containerStore.containers.find((c) => c.id === containerStore.activeContainerID) || null
)

const activeContainerName = computed(() => {
  const c = activeContainer.value
  if (!c) return t('runtime.noActiveContainer')
  return c.name || c.id.substring(0, 12)
})
</script>

<template>
  <section class="runtime-inspector">
    <header class="runtime-head">
      <div class="runtime-title-block">
        <span class="runtime-kicker">{{ t('runtime.title') }}</span>
        <span class="runtime-active" :title="activeContainer?.id || ''">
          <NIcon size="14"><RadioButtonOn /></NIcon>
          {{ activeContainerName }}
        </span>
      </div>
      <div class="runtime-tabs" role="tablist" :aria-label="t('runtime.title')">
        <button
          type="button"
          :class="['runtime-tab', { active: activeTab === 'containers' }]"
          role="tab"
          :aria-selected="activeTab === 'containers'"
          :aria-controls="'runtime-containers'"
          @click="activeTab = 'containers'"
        >
          <NIcon size="14"><Cube /></NIcon>
          {{ t('runtime.containers') }}
        </button>
        <button
          type="button"
          :class="['runtime-tab', { active: activeTab === 'terminal' }]"
          role="tab"
          :aria-selected="activeTab === 'terminal'"
          :aria-controls="'runtime-terminal'"
          @click="activeTab = 'terminal'"
        >
          <NIcon size="14"><Terminal /></NIcon>
          {{ t('runtime.terminal') }}
        </button>
      </div>
    </header>

    <div class="runtime-body">
      <ContainerPanel
        v-show="activeTab === 'containers'"
        id="runtime-containers"
        role="tabpanel"
        class="runtime-pane"
      />
      <TerminalPanel
        v-show="activeTab === 'terminal'"
        id="runtime-terminal"
        role="tabpanel"
        class="runtime-pane"
      />
    </div>
  </section>
</template>

<style scoped>
.runtime-inspector {
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: transparent;
  display: flex;
  flex-direction: column;
}

.runtime-head {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  padding: 12px 12px 10px;
  border-bottom: 1px solid var(--border-subtle);
  background: rgba(2, 6, 23, 0.32);
}

.runtime-title-block {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.runtime-kicker {
  color: var(--text-faint);
  font-family: var(--font-brand);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  letter-spacing: 0.7px;
  text-transform: uppercase;
}

.runtime-active {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-active .n-icon {
  color: var(--accent-emerald);
  flex-shrink: 0;
}

.runtime-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-xs);
}

.runtime-tab {
  height: 30px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-deepest);
  color: var(--text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-xs);
  font-size: var(--fs-xs);
  font-family: var(--font-sans);
  transition: background var(--transition-ui), color var(--transition-ui), border-color var(--transition-ui);
}

.runtime-tab:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.runtime-tab.active {
  color: var(--accent-cyan);
  background: color-mix(in srgb, var(--accent-cyan) 10%, var(--bg-elevated));
  border-color: rgba(34, 211, 238, 0.3);
}

.runtime-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.runtime-pane {
  height: 100%;
  min-height: 0;
}
</style>
