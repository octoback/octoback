<template>
  <plugin-selector :model-value="modelValue" plugin-type="repository" placeholder="请选择存储仓库" dialog-title="选择存储仓库" @update:model-value="emit('update:modelValue', $event)" />
</template>

<script setup lang="ts">
defineOptions({
  name: "RepositorySelector"
});

defineProps<{ modelValue?: string | number | null }>();
const emit = defineEmits<{ (event: "update:modelValue", value: number | null): void }>();
</script>
