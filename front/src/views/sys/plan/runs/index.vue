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
    background: #111315;
    color: #e8eaed;
  }

  .ant-modal-header {
    background: #111315;
    border-bottom-color: #2d3238;
  }

  .ant-modal-title,
  .ant-modal-close {
    color: #e8eaed;
  }

  .ant-modal-body {
    padding: 0;
  }

  .ant-modal-confirm-body-wrapper {
    margin: 0;
  }
}

.backup-log-modal-content {
  background: #111315;
  padding: 0 18px 18px;
}

.backup-log-summary {
  border-bottom: 1px solid #2d3238;
  color: #c8cdd3;
  padding: 14px 0;
}

.backup-log-summary-title {
  color: #f0f2f4;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}

.backup-log-summary-grid {
  display: grid;
  gap: 6px 18px;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 12px;
}

.backup-log-summary-empty {
  color: #7f8790;
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
