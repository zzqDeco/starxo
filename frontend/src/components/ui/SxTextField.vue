<script setup lang="ts">
withDefaults(defineProps<{
  label: string
  modelValue?: string
  placeholder?: string
  state?: 'default' | 'focus' | 'error' | 'disabled'
  error?: string
  type?: string
}>(), {
  modelValue: '',
  placeholder: '',
  state: 'default',
  error: '',
  type: 'text',
})

defineEmits<{ (e: 'update:modelValue', value: string): void }>()
</script>

<template>
  <label class="sx-text-field" :class="`state-${state}`">
    <span class="sx-text-field__label">{{ label }}</span>
    <span class="sx-text-field__control">
      <input
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="state === 'disabled'"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      />
      <span v-if="state === 'error' && error" class="sx-text-field__error">{{ error }}</span>
    </span>
  </label>
</template>

<style scoped>
.sx-text-field {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  align-items: start;
  gap: 14px;
  width: 100%;
}
.sx-text-field__label {
  padding-top: 8px;
  color: var(--sx-text-secondary);
  font: 500 12px/16px var(--sx-font-sans);
}
.sx-text-field__control {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
input {
  width: 100%;
  height: 32px;
  border: 1px solid var(--sx-separator-strong);
  border-radius: var(--sx-radius-8);
  background: var(--sx-control-bg);
  color: var(--sx-text-primary);
  padding: 0 12px;
  font: 400 12px/16px var(--sx-font-sans);
  outline: none;
}
input:focus,
.state-focus input {
  border-color: var(--sx-focus);
  box-shadow: 0 0 0 2px var(--sx-accent-bg);
}
.state-error input {
  border-color: var(--sx-danger);
  background: var(--sx-danger-bg);
}
.state-disabled {
  opacity: 0.55;
}
.sx-text-field__error {
  color: var(--sx-danger);
  font: 400 10px/13px var(--sx-font-sans);
}
</style>
