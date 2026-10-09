<template>
  <a-button class="plugin-test-button" type="primary" :loading="loading" :disabled="!form?.id" @click="test">
    {{ text }}
  </a-button>
</template>

<script setup lang="ts">
defineOptions({
  name: "PluginTestButton"
});

import { message } from "ant-design-vue";
import { inject, ref } from "vue";
import { executePluginInstanceAction } from "/src/views/sys/plugin/plugin-api";

const props = withDefaults(defineProps<{ action: string; text?: string }>(), { text: "测试" });
const loading = ref(false);
const getScope = inject<() => { form?: Record<string, any> }>("get:scope");
const form = getScope?.()?.form;

async function test() {
  if (!form?.id || loading.value) {
    return;
  }
  loading.value = true;
  try {
    await executePluginInstanceAction(form.id, props.action, form);
    message.success("测试成功");
  } catch (error: any) {
    message.error(error?.message || "测试失败");
  } finally {
    loading.value = false;
  }
}
</script>

<style lang="less">
.plugin-test-button {
  min-width: 96px;
}
</style>
