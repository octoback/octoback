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
}

.restore-task-log-modal {
  .ant-modal-content {
    background: #fff;
    color: rgba(0, 0, 0, 0.88);
  }
}

.restore-task-log-terminal {
  background: #080a0c;
  border: 1px solid #252a30;
  border-radius: 4px;
  box-sizing: border-box;
  color: #d7dbe0;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 12px;
  line-height: 1.65;
  max-height: 560px;
  overflow: auto;
  padding: 14px;
  white-space: pre-wrap;
  word-break: break-word;
}

.restore-task-log-line {
  min-height: 20px;
}

.restore-task-log-line-plain,
.restore-task-log-message {
  color: #d7dbe0;
}

.restore-task-log-time,
.restore-task-log-level {
  display: inline-block;
  margin-right: 10px;
}

.restore-task-log-level-info .restore-task-log-time,
.restore-task-log-level-info .restore-task-log-level {
  color: #62d991;
}

.restore-task-log-level-warn .restore-task-log-time,
.restore-task-log-level-warn .restore-task-log-level {
  color: #f2c866;
}

.restore-task-log-level-error .restore-task-log-time,
.restore-task-log-level-error .restore-task-log-level {
  color: #ff7474;
}
</style>
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
