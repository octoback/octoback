<template>
  <fs-page class="restore-task-page">
    <template #header
      ><div class="title">恢复任务 <span class="sub">查看恢复进度、结果与执行日志</span></div></template
    >
    <fs-crud ref="crudRef" v-bind="crudBinding" />
  </fs-page>
</template>
<script setup lang="ts">
import { onMounted } from "vue";
import { useFs } from "@fast-crud/fast-crud";
import createCrudOptions from "./crud";
const { crudBinding, crudRef, crudExpose } = useFs({ createCrudOptions });
onMounted(() => crudExpose.doRefresh());
defineOptions({ name: "RestoreTask" });
</script>
<style lang="less">
.restore-task-page {
  .restore-task-snapshot-id {
    display: block;
    max-width: 160px;
    overflow: hidden;
    white-space: nowrap;
  }
}
</style>
