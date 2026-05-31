<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  description?: string
}>(), {
  title: '',
  description: '',
})
</script>

<template>
  <section class="sx-modal-surface">
    <header v-if="title || description" class="sx-modal-head">
      <strong v-if="title">{{ title }}</strong>
      <p v-if="description">{{ description }}</p>
    </header>
    <div class="sx-modal-body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.sx-modal-surface {
  overflow: hidden;
  border: 1px solid var(--sx-separator);
  border-radius: var(--sx-radius-12);
  background: var(--sx-elevated-bg);
  box-shadow: var(--sx-shadow-2);
}

.sx-modal-head {
  padding: 18px 20px 12px;
  border-bottom: 1px solid var(--sx-separator);
}

.sx-modal-head strong {
  color: var(--sx-text-primary);
  font: 600 16px/22px var(--sx-font-sans);
}

.sx-modal-head p {
  margin-top: 4px;
  color: var(--sx-text-tertiary);
  font: 400 12px/18px var(--sx-font-sans);
}

.sx-modal-body {
  padding: 18px 20px;
}
</style>
