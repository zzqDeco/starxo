<script lang="ts" setup>
import { computed, nextTick, ref, watch } from 'vue'
import { NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useFocusTrap } from '@/composables/useFocusTrap'
import { comboLabel, type KeyCombo } from '@/composables/useKeybinds'
import { useSessionStore } from '@/stores/sessionStore'
import { useChatStore } from '@/stores/chatStore'
import { useConnectionStore } from '@/stores/connectionStore'
import { useUiFeedback } from '@/composables/useUiFeedback'
import { SetMode } from '../../../wailsjs/go/service/ChatService'
import {
  Add, Settings, SwapHorizontal, ChatbubbleEllipses, Flash, Power, Search, Close,
  FolderOpen,
} from '@vicons/ionicons5'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'open-settings'): void
  (e: 'open-workspace'): void
}>()

const { t } = useI18n()
const sessionStore = useSessionStore()
const chatStore = useChatStore()
const connectionStore = useConnectionStore()
const feedback = useUiFeedback()

const query = ref('')
const cursor = ref(0)
const dialogRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLElement | null>(null)
const active = computed(() => props.show)
useFocusTrap(dialogRef, active)

type Command = {
  id: string
  title: string
  hint?: string
  icon: any
  combo?: KeyCombo
  group: 'action' | 'session' | 'mode'
  run: () => void | Promise<void>
}

const baseCommands = computed<Command[]>(() => {
  const cmds: Command[] = [
    {
      id: 'new-session',
      title: t('palette.newSession'),
      hint: t('palette.hintNewSession'),
      icon: Add,
      combo: { key: 'n', meta: true },
      group: 'action',
      run: async () => {
        try {
          await sessionStore.createSession()
          feedback.success(t('feedback.sessionCreated'))
        } catch (e) {
          feedback.error(t('feedback.actions.createSession'), e)
        }
      },
    },
    {
      id: 'open-settings',
      title: t('palette.openSettings'),
      hint: t('settings.title'),
      icon: Settings,
      combo: { key: ',', meta: true },
      group: 'action',
      run: () => emit('open-settings'),
    },
    {
      id: 'open-workspace',
      title: t('palette.openWorkspace'),
      hint: t('workspace.drawerTitle'),
      icon: FolderOpen,
      group: 'action',
      run: () => emit('open-workspace'),
    },
    {
      id: 'toggle-mode',
      title: chatStore.agentMode === 'plan' ? t('palette.switchToDefault') : t('palette.switchToPlan'),
      hint: t('chat.modeLabel'),
      icon: SwapHorizontal,
      group: 'mode',
      run: async () => {
        const next = chatStore.agentMode === 'plan' ? 'default' : 'plan'
        try {
          await SetMode(next)
          chatStore.setMode(next)
          feedback.success(t('feedback.modeSwitched', { mode: next === 'plan' ? t('chat.modePlan') : t('chat.modeDefault') }))
        } catch (e) {
          feedback.error(t('feedback.actions.switchMode'), e)
        }
      },
    },
  ]
  if (!connectionStore.sshConnected) {
    cmds.push({
      id: 'connect-ssh',
      title: t('palette.connectSSH'),
      icon: Power,
      group: 'action',
      run: () => connectionStore.connect(),
    })
  }
  return cmds
})

const sessionCommands = computed<Command[]>(() => {
  return sessionStore.sessions.slice(0, 20).map((s, idx) => ({
    id: `switch-${s.id}`,
    title: s.title || t('sidebar.untitled'),
    hint: idx < 9 ? comboLabel({ key: String(idx + 1), meta: true }) : undefined,
    icon: ChatbubbleEllipses,
    combo: idx < 9 ? { key: String(idx + 1), meta: true } : undefined,
    group: 'session',
    run: async () => {
      try {
        await sessionStore.switchSession(s.id)
      } catch (e) {
        feedback.error(t('feedback.actions.switchSession'), e)
      }
    },
  }))
})

const allCommands = computed<Command[]>(() => [...baseCommands.value, ...sessionCommands.value])

const filtered = computed<Command[]>(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return allCommands.value
  return allCommands.value.filter(c => c.title.toLowerCase().includes(q) || (c.hint?.toLowerCase().includes(q) ?? false))
})

const groups = computed(() => {
  const byGroup: Record<string, Command[]> = {}
  for (const c of filtered.value) {
    byGroup[c.group] ??= []
    byGroup[c.group].push(c)
  }
  return byGroup
})

const flatList = computed<Command[]>(() => filtered.value)

const groupLabel: Record<string, string> = {
  action: 'palette.groupActions',
  session: 'palette.groupSessions',
  mode: 'palette.groupModes',
}

watch(() => props.show, (v) => {
  if (v) {
    query.value = ''
    cursor.value = 0
    nextTick(() => inputRef.value?.focus())
  }
}, { immediate: true })

watch(query, () => {
  cursor.value = 0
})

function close() {
  emit('update:show', false)
}

async function runCommand(cmd: Command) {
  close()
  await cmd.run()
}

function scrollIntoView() {
  nextTick(() => {
    const el = listRef.value?.querySelector<HTMLElement>(`[data-idx="${cursor.value}"]`)
    if (el) el.scrollIntoView({ block: 'nearest' })
  })
}

function onKeydown(e: KeyboardEvent) {
  const list = flatList.value
  if (list.length === 0) return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    cursor.value = (cursor.value + 1) % list.length
    scrollIntoView()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    cursor.value = (cursor.value - 1 + list.length) % list.length
    scrollIntoView()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const cmd = list[cursor.value]
    if (cmd) runCommand(cmd)
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}

function indexOf(cmd: Command): number {
  return flatList.value.indexOf(cmd)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="palette">
      <div
        v-if="show"
        class="palette-overlay"
        @mousedown.self="close"
        @keydown="onKeydown"
      >
        <div
          ref="dialogRef"
          class="palette-dialog"
          role="dialog"
          aria-modal="true"
          :aria-label="t('palette.title')"
          tabindex="-1"
        >
          <div class="palette-search">
            <NIcon size="16" class="search-icon"><Search /></NIcon>
            <input
              ref="inputRef"
              v-model="query"
              type="text"
              class="palette-input"
              :placeholder="t('palette.placeholder')"
              :aria-label="t('palette.placeholder')"
              autocomplete="off"
              spellcheck="false"
            />
            <button
              type="button"
              class="palette-close"
              :aria-label="t('common.cancel')"
              @click="close"
            >
              <NIcon size="14"><Close /></NIcon>
            </button>
          </div>

          <div ref="listRef" class="palette-list" role="listbox" :aria-activedescendant="flatList[cursor] ? `palette-item-${cursor}` : undefined">
            <template v-if="flatList.length === 0">
              <div class="palette-empty">
                <NIcon size="20"><Flash /></NIcon>
                <span>{{ t('palette.empty') }}</span>
              </div>
            </template>

            <template v-else v-for="(items, key) in groups" :key="key">
              <div class="group-label">{{ t(groupLabel[key] || key) }}</div>
              <button
                v-for="cmd in items"
                :key="cmd.id"
                :id="`palette-item-${indexOf(cmd)}`"
                type="button"
                role="option"
                :data-idx="indexOf(cmd)"
                :class="['palette-item', { active: indexOf(cmd) === cursor }]"
                :aria-selected="indexOf(cmd) === cursor"
                @click="runCommand(cmd)"
                @mouseenter="cursor = indexOf(cmd)"
              >
                <span class="item-icon"><NIcon size="16"><component :is="cmd.icon" /></NIcon></span>
                <span class="item-title">{{ cmd.title }}</span>
                <span v-if="cmd.hint" class="item-hint">{{ cmd.hint }}</span>
                <span v-else-if="cmd.combo" class="item-kbd">{{ comboLabel(cmd.combo) }}</span>
              </button>
            </template>
          </div>

          <footer class="palette-footer">
            <span><kbd>↑↓</kbd> {{ t('palette.navigate') }}</span>
            <span><kbd>Enter</kbd> {{ t('palette.run') }}</span>
            <span><kbd>Esc</kbd> {{ t('palette.close') }}</span>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.palette-overlay {
  position: fixed;
  inset: 0;
  background: rgba(2, 6, 23, 0.68);
  backdrop-filter: blur(8px);
  z-index: 2200;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 12vh;
}

.palette-dialog {
  width: min(560px, 92vw);
  max-height: min(520px, 72vh);
  display: flex;
  flex-direction: column;
  background: color-mix(in srgb, var(--bg-surface) 94%, black);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--elev-3);
  overflow: hidden;
}

.palette-search {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border-subtle);
  background: rgba(2, 6, 23, 0.42);
  flex-shrink: 0;
}

.search-icon {
  color: var(--text-faint);
  flex-shrink: 0;
}

.palette-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: var(--fs-md);
  font-family: inherit;
}

.palette-input::placeholder {
  color: var(--text-faint);
}

.palette-close {
  background: transparent;
  border: none;
  color: var(--text-faint);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background var(--transition-ui), color var(--transition-ui);
}

.palette-close:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.palette-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px 6px 10px;
  min-height: 0;
}

.group-label {
  font-size: var(--fs-2xs);
  text-transform: uppercase;
  letter-spacing: 0.8px;
  color: var(--text-faint);
  padding: 10px 12px 4px;
  font-weight: var(--fw-semibold);
}

.palette-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 12px;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  text-align: left;
  font-size: var(--fs-sm);
  transition: background var(--transition-ui), color var(--transition-ui);
}

.palette-item.active {
  background: color-mix(in srgb, var(--accent-cyan) 10%, var(--bg-hover));
  color: var(--text-primary);
}

.item-icon {
  display: flex;
  align-items: center;
  color: var(--accent-cyan);
  flex-shrink: 0;
}

.item-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-hint {
  font-size: var(--fs-2xs);
  color: var(--text-faint);
  flex-shrink: 0;
}

.item-kbd {
  font-family: var(--font-mono);
  font-size: var(--fs-2xs);
  color: var(--text-muted);
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 2px 6px;
  flex-shrink: 0;
}

.palette-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 12px;
  color: var(--text-faint);
  font-size: var(--fs-sm);
}

.palette-footer {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px 14px;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-surface);
  font-size: var(--fs-2xs);
  color: var(--text-faint);
  flex-shrink: 0;
}

.palette-footer kbd {
  font-family: var(--font-mono);
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 1px 5px;
  margin-right: 4px;
}

.palette-enter-active,
.palette-leave-active {
  transition: opacity 160ms var(--ease-out);
}

.palette-enter-active .palette-dialog,
.palette-leave-active .palette-dialog {
  transition: transform 160ms var(--ease-out);
}

.palette-enter-from,
.palette-leave-to {
  opacity: 0;
}

.palette-enter-from .palette-dialog,
.palette-leave-to .palette-dialog {
  transform: translateY(-8px);
}

@media (prefers-reduced-motion: reduce) {
  .palette-enter-active,
  .palette-leave-active,
  .palette-enter-active .palette-dialog,
  .palette-leave-active .palette-dialog {
    transition: none;
  }
}
</style>
