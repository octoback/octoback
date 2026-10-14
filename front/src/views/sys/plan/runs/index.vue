<template>
  <fs-page class="backup-log-page">
    <template #header><div class="title">备份日志</div></template>
    <fs-crud ref="crudRef" v-bind="crudBinding" />
  </fs-page>
</template>
<script setup lang="ts">
import { onMounted } from "vue";
import { useFs } from "@fast-crud/fast-crud";
import createCrudOptions from "./crud";
const { crudBinding, crudRef, crudExpose } = useFs({ createCrudOptions });
onMounted(() => crudExpose.doRefresh());
defineOptions({ name: "BackupLog" });
</script>
<style lang="less">
.backup-log-page {
}

.backup-log-modal {
  .ant-modal-content {
    background: #fff;
    color: rgba(0, 0, 0, 0.88);
  }
}

.backup-log-modal-content {
  padding: 0;
}

.backup-log-summary-dialog {
  max-height: 560px;
  overflow: auto;
  padding: 4px 2px;
}

.backup-log-summary-title {
  color: rgba(0, 0, 0, 0.88);
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 14px;
}

.backup-log-summary-grid {
  border-top: 1px solid #f0f0f0;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.backup-log-summary-item {
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
  padding: 10px 12px;
}

.backup-log-summary-snapshot {
  grid-column: span 2;
}

.backup-log-summary-key {
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}

.backup-log-summary-value {
  color: rgba(0, 0, 0, 0.88);
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 13px;
  overflow-wrap: anywhere;
}

.backup-log-summary-empty {
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}

.backup-log-terminal {
  background: #080a0c;
  border: 1px solid #252a30;
  border-radius: 4px;
  box-sizing: border-box;
  color: #d7dbe0;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 12px;
  line-height: 1.65;
  margin-top: 14px;
  max-height: 560px;
  overflow: auto;
  padding: 14px;
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 640px) {
  .backup-log-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .backup-log-summary-snapshot {
    grid-column: span 2;
  }
}

.backup-log-line {
  min-height: 20px;
}

.backup-log-line-plain,
.backup-log-message {
  color: #d7dbe0;
}

.backup-log-time,
.backup-log-level {
  display: inline-block;
  margin-right: 10px;
}

.backup-log-level-info .backup-log-time,
.backup-log-level-info .backup-log-level {
  color: #62d991;
}

.backup-log-level-warn .backup-log-time,
.backup-log-level-warn .backup-log-level {
  color: #f2c866;
}

.backup-log-level-error .backup-log-time,
.backup-log-level-error .backup-log-level {
  color: #ff7474;
}
</style>
