<script lang="ts" setup>
import { ref, onUnmounted, computed } from 'vue'

const props = withDefaults(defineProps<{
  direction?: 'horizontal' | 'vertical'
  minSize?: number
  maxSize?: number
  defaultSize: number
  storageKey?: string
  reverse?: boolean
}>(), {
  direction: 'horizontal',
  minSize: 180,
  maxSize: 600,
  reverse: false,
})

const emit = defineEmits<{
  (e: 'update:size', value: number): void
}>()

const isDragging = ref(false)
let startPos = 0
let startSize = 0
let currentSize = props.defaultSize

// Restore from localStorage
if (props.storageKey) {
  const saved = localStorage.getItem(props.storageKey)
  if (saved) {
    const parsed = Number(saved)
    if (!isNaN(parsed) && parsed >= props.minSize && parsed <= props.maxSize) {
      currentSize = parsed
      emit('update:size', currentSize)
    }
  }
}

const cursorClass = computed(() =>
  props.direction === 'horizontal' ? 'col-resize' : 'row-resize'
)

function onMouseDown(e: MouseEvent) {
  e.preventDefault()
  isDragging.value = true
  startPos = props.direction === 'horizontal' ? e.clientX : e.clientY
  startSize = currentSize

  document.body.style.cursor = cursorClass.value
  document.body.style.userSelect = 'none'

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

function onMouseMove(e: MouseEvent) {
  if (!isDragging.value) return

  const currentPos = props.direction === 'horizontal' ? e.clientX : e.clientY
  const delta = currentPos - startPos
  const newSize = props.reverse
    ? Math.min(props.maxSize, Math.max(props.minSize, startSize - delta))
    : Math.min(props.maxSize, Math.max(props.minSize, startSize + delta))

  currentSize = newSize
  emit('update:size', newSize)
}

function onMouseUp() {
  isDragging.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''

  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)

  // Persist to localStorage
  if (props.storageKey) {
    localStorage.setItem(props.storageKey, String(currentSize))
  }
}

function onDoubleClick() {
  currentSize = props.defaultSize
  emit('update:size', props.defaultSize)
  if (props.storageKey) {
    localStorage.setItem(props.storageKey, String(props.defaultSize))
  }
}

onUnmounted(() => {
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
})
</script>

<template>
  <div
    :class="['split-handle', direction, { dragging: isDragging }]"
    @mousedown="onMouseDown"
    @dblclick="onDoubleClick"
  >
    <div class="split-handle-line" />
  </div>
</template>

<style scoped>
.split-handle {
  position: relative;
  flex-shrink: 0;
  z-index: var(--z-splitter, 10);
  background: transparent;
  transition: background 150ms ease-out;
}

.split-handle.horizontal {
  width: var(--splitter-width, 4px);
  cursor: col-resize;
}

.split-handle.vertical {
  height: var(--splitter-width, 4px);
  cursor: row-resize;
}

.split-handle:hover,
.split-handle.dragging {
  background: var(--splitter-hover, var(--accent-cyan-dim));
}

.split-handle-line {
  position: absolute;
  opacity: 0;
  transition: opacity 150ms ease-out;
  background: var(--splitter-active, var(--accent-cyan));
}

.split-handle.horizontal .split-handle-line {
  top: 0;
  bottom: 0;
  left: 1px;
  width: 2px;
}

.split-handle.vertical .split-handle-line {
  left: 0;
  right: 0;
  top: 1px;
  height: 2px;
}

.split-handle:hover .split-handle-line,
.split-handle.dragging .split-handle-line {
  opacity: 1;
}
</style>
