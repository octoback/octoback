<template>
  <plugin-selector :model-value="modelValue" plugin-type="source" placeholder="请选择数据资源" dialog-title="选择数据资源" @update:model-value="emit('update:modelValue', $event)" />
</template>

<script setup lang="ts">
defineOptions({
  name: "SourceSelector"
});

defineProps<{ modelValue?: string | number | null }>();
const emit = defineEmits<{ (event: "update:modelValue", value: number | null): void }>();
</script>
