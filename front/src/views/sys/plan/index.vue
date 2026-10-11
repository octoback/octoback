<template>
  <fs-page class="component-plan">
    <template #header>
      <div class="title">备份计划 <span class="sub">管理数据资源、调度和存储仓库</span></div>
    </template>
    <fs-crud ref="crudRef" v-bind="crudBinding" />
  </fs-page>
</template>
<script setup lang="ts">
import { onMounted } from "vue";
import { useFs } from "@fast-crud/fast-crud";
import createCrudOptions from "./crud";
const { crudBinding, crudRef, crudExpose } = useFs({ createCrudOptions });
function goToFlow() {
  window.location.hash = "#/sys/plan/flow";
}
onMounted(() => crudExpose.doRefresh());
defineOptions({
  name: "BackupPlan"
});
</script>
<style lang="less">
.component-plan {
}
</style>
