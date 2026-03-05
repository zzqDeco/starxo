<script lang="ts" setup>
import ContainerPanel from './ContainerPanel.vue'
</script>

<template>
  <section class="container-dock">
    <ContainerPanel />
  </section>
</template>

<style scoped>
.container-dock {
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--bg-surface);
}
</style>
