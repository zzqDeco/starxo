<script lang="ts" setup>
import { computed } from 'vue'
import { NCollapse, NCollapseItem } from 'naive-ui'
import { useChatStore } from '@/stores/chatStore'

import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const chatStore = useChatStore()

const steps = computed(() => chatStore.planSteps)
const hasSteps = computed(() => steps.value.length > 0)

const completedCount = computed(() => steps.value.filter(s => s.status === 'done').length)
const totalCount = computed(() => steps.value.length)
const progressPercent = computed(() =>
  totalCount.value > 0 ? Math.round((completedCount.value / totalCount.value) * 100) : 0
)

function statusIcon(status: string): string {
  switch (status) {
    case 'done': return '\u2713'
    case 'doing': return '\u25B6'
    case 'failed': return '\u2717'
    case 'skipped': return '\u2014'
    default: return '\u25CB'
  }
}

function statusClass(status: string): string {
  return `step-status-${status}`
}
</script>

<template>
  <div v-if="hasSteps" class="plan-panel">
    <NCollapse :default-expanded-names="['plan']">
      <NCollapseItem :title="t('plan.title')" name="plan">
        <template #header-extra>
          <span class="plan-progress">{{ completedCount }}/{{ totalCount }} ({{ progressPercent }}%)</span>
        </template>

        <div class="plan-steps">
          <div
            v-for="step in steps"
            :key="step.taskId"
            class="plan-step"
            :class="statusClass(step.status)"
          >
            <span class="step-icon" :class="statusClass(step.status)">
              {{ statusIcon(step.status) }}
            </span>
            <div class="step-content">
              <span class="step-desc">{{ step.desc }}</span>
              <span v-if="step.execResult" class="step-result">{{ step.execResult }}</span>
            </div>
          </div>
        </div>
      </NCollapseItem>
    </NCollapse>
  </div>
</template>

<style scoped>
.plan-panel {
  max-width: 800px;
  margin: 0 auto 12px;
  padding: 0 24px;
}

.plan-panel :deep(.n-collapse-item__header) {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.plan-panel :deep(.n-collapse-item__content-inner) {
  padding: 4px 0 0 !important;
}

.plan-progress {
  font-size: 11px;
  color: var(--accent-cyan);
  font-weight: 600;
  font-family: var(--font-mono);
}

.plan-steps {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.plan-step {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  transition: background 200ms ease;
}

.plan-step:hover {
  background: var(--bg-surface);
}

.step-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
  border-radius: 50%;
  margin-top: 1px;
}

.step-icon.step-status-done {
  color: #34d399;
  background: rgba(52, 211, 153, 0.12);
}

.step-icon.step-status-doing {
  color: var(--accent-cyan);
  background: rgba(34, 211, 238, 0.12);
  animation: pulse 1.5s ease-in-out infinite;
}

.step-icon.step-status-failed {
  color: #f87171;
  background: rgba(248, 113, 113, 0.12);
}

.step-icon.step-status-skipped {
  color: var(--text-muted);
  background: var(--bg-surface);
}

.step-icon.step-status-todo {
  color: var(--text-faint);
  background: var(--bg-surface);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.step-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.step-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.4;
}

.step-status-done .step-desc {
  color: var(--text-muted);
}

.step-status-doing .step-desc {
  color: var(--text-primary);
  font-weight: 500;
}

.step-result {
  font-size: 11px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  overflow-wrap: break-word;
  word-break: break-word;
  max-width: 500px;
  line-height: 1.4;
}
</style>
