<script lang="ts" setup>
import { computed } from 'vue'
import { NIcon } from 'naive-ui'
import { RadioButtonOn } from '@vicons/ionicons5'
import ContainerPanel from './ContainerPanel.vue'
import { useContainerStore } from '@/stores/containerStore'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const containerStore = useContainerStore()

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
    </header>

    <div class="runtime-body">
      <ContainerPanel
        id="runtime-containers"
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
  padding: 10px 12px 9px;
  border-bottom: 1px solid var(--border-subtle);
  background: color-mix(in srgb, var(--platform-bg-toolbar) 86%, transparent);
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
  font-weight: var(--fw-medium);
  letter-spacing: 0;
  text-transform: none;
}

.runtime-active {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--text-primary);
  font-family: var(--font-sans);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-active .n-icon {
  color: var(--text-faint);
  flex-shrink: 0;
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
