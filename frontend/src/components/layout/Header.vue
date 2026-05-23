<script lang="ts" setup>
import { computed } from 'vue'
import { NButton, NTooltip, NIcon } from 'naive-ui'
import { Settings, FolderOpen, Search, Flash, List } from '@vicons/ionicons5'
import ConnectionStatus from '@/components/status/ConnectionStatus.vue'
import { useI18n } from 'vue-i18n'
import { useChatStore } from '@/stores/chatStore'
import { useSessionStore } from '@/stores/sessionStore'

const { t, locale } = useI18n()
const chatStore = useChatStore()
const sessionStore = useSessionStore()

const activeSessionTitle = computed(() => sessionStore.activeSession?.title || t('sidebar.untitled'))
const modeLabel = computed(() => chatStore.agentMode === 'plan' ? t('chat.modePlan') : t('chat.modeDefault'))

defineProps<{
  workspaceDrawerVisible: boolean
  runtimeTasksVisible: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-settings'): void
  (e: 'toggle-workspace-drawer'): void
  (e: 'toggle-runtime-tasks'): void
  (e: 'open-command-palette'): void
}>()

function toggleLocale() {
  locale.value = locale.value === 'en' ? 'zh' : 'en'
  localStorage.setItem('locale', locale.value)
}
</script>

<template>
  <header class="app-header wails-drag" role="banner">
    <div class="header-left">
      <div class="app-title" aria-label="Starxo">
        <span class="title-icon" aria-hidden="true"><NIcon size="14"><Flash /></NIcon></span>
        <span class="title-text">{{ t('header.title') }}</span>
      </div>
    </div>

    <div class="header-center">
      <button
        type="button"
        class="command-trigger"
        :aria-label="t('header.commandPalette')"
        @click="emit('open-command-palette')"
      >
        <NIcon size="14" class="command-icon"><Search /></NIcon>
        <span class="command-copy">
          <span class="command-main">{{ activeSessionTitle }}</span>
          <span class="command-sub">{{ modeLabel }} · {{ t('header.commandPlaceholder') }}</span>
        </span>
        <kbd class="command-kbd" aria-hidden="true">⌘K</kbd>
      </button>
    </div>

    <div class="header-right">
      <ConnectionStatus />

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <NButton
            quaternary
            circle
            size="small"
            class="header-btn"
            :aria-label="runtimeTasksVisible ? t('header.runtimeTasksClose') : t('header.runtimeTasksOpen')"
            @click="emit('toggle-runtime-tasks')"
          >
            <template #icon>
              <List />
            </template>
          </NButton>
        </template>
        {{ runtimeTasksVisible ? t('header.runtimeTasksClose') : t('header.runtimeTasksOpen') }}
      </NTooltip>

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <NButton
            quaternary
            circle
            size="small"
            class="header-btn"
            :aria-label="workspaceDrawerVisible ? t('header.workspaceClose') : t('header.workspaceOpen')"
            @click="emit('toggle-workspace-drawer')"
          >
            <template #icon>
              <FolderOpen />
            </template>
          </NButton>
        </template>
        {{ workspaceDrawerVisible ? t('header.workspaceClose') : t('header.workspaceOpen') }}
      </NTooltip>

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <NButton
            quaternary
            size="small"
            class="header-btn lang-btn"
            :aria-label="locale === 'en' ? '切换到中文' : 'Switch to English'"
            @click="toggleLocale"
          >
            {{ locale === 'en' ? '中' : 'EN' }}
          </NButton>
        </template>
        {{ locale === 'en' ? '切换到中文' : 'Switch to English' }}
      </NTooltip>

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <NButton
            quaternary
            circle
            size="small"
            class="header-btn"
            :aria-label="t('header.settings')"
            @click="emit('toggle-settings')"
          >
            <template #icon>
              <Settings />
            </template>
          </NButton>
        </template>
        {{ t('header.settings') }}
      </NTooltip>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 var(--space-lg);
  background: var(--platform-bg-toolbar);
  backdrop-filter: blur(24px) saturate(1.35);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
  z-index: var(--z-sticky, 20);
  position: relative;
}

:global(:root[data-platform="macos"] .app-header){
  height: 52px;
  padding-left: 76px;
  padding-right: 12px;
  background: color-mix(in srgb, var(--platform-bg-toolbar) 82%, transparent);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.app-title {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  user-select: none;
}

.title-icon {
  color: var(--platform-accent);
  display: flex;
}

.title-text {
  font-family: var(--font-brand);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--text-primary);
  letter-spacing: 0;
}

.header-center {
  flex: 1;
  display: flex;
  justify-content: center;
  min-width: 0;
  padding: 0 var(--space-lg);
}

.command-trigger {
  width: min(520px, 100%);
  height: 38px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--platform-bg-raised) 72%, transparent);
  color: var(--text-secondary);
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-sm);
  padding: 0 var(--space-md);
  text-align: left;
  transition: border-color var(--transition-ui), background var(--transition-ui), box-shadow var(--transition-ui);
}

:global(:root[data-platform="macos"] .command-trigger){
  height: 34px;
  max-width: 520px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--platform-bg-raised) 72%, transparent);
  box-shadow: inset 0 0 0 0.5px color-mix(in srgb, var(--border-subtle) 80%, transparent);
}

.command-trigger:hover,
.command-trigger:focus-visible {
  background: var(--platform-bg-raised);
  border-color: color-mix(in srgb, var(--platform-accent) 36%, var(--border-subtle));
  box-shadow: var(--platform-shadow-1);
}

.command-icon {
  color: var(--text-faint);
  flex-shrink: 0;
}

.command-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.command-main,
.command-sub {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-main {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--text-primary);
}

:global(:root[data-platform="macos"] .command-main){
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

:global(:root[data-platform="macos"] .command-sub){
  font-size: 10.5px;
}

.command-sub {
  font-size: var(--fs-2xs);
  color: var(--text-faint);
}

.command-kbd {
  font-family: var(--font-mono);
  font-size: var(--fs-2xs);
  color: var(--text-muted);
  background: color-mix(in srgb, var(--platform-bg-toolbar) 78%, transparent);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 2px 6px;
  line-height: 1.2;
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  flex-shrink: 0;
}

.header-btn {
  color: var(--text-muted) !important;
  transition: color var(--transition-ui), background var(--transition-ui) !important;
}

.header-btn:hover {
  color: var(--text-primary) !important;
}

.lang-btn {
  font-family: var(--font-sans);
  font-size: var(--fs-xs) !important;
  font-weight: var(--fw-bold) !important;
  letter-spacing: 0;
  min-width: 32px;
}

@media (max-width: 860px) {
  .command-sub {
    display: none;
  }

  .command-trigger {
    height: 34px;
  }
}

@media (max-width: 640px) {
  .title-text,
  .command-kbd {
    display: none;
  }

  .header-center {
    padding: 0 var(--space-sm);
  }
}
</style>
