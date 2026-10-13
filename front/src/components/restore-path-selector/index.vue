<template>
  <div class="restore-path-selector">
    <PluginPathSelector :model-value="modelValue" :instance-id="instanceId" :access-id="accessId" :action="action" :modal-title="modalTitle" :placeholder="placeholder" @update:model-value="onTargetPathChange" />
    <div class="restore-path-selector__preview" aria-live="polite">
      <div class="restore-path-selector__preview-heading">恢复目录预览</div>
      <div v-if="!selectedTargetPath" class="restore-path-selector__empty">请选择恢复目录</div>
      <div v-else-if="!sourcePaths.length" class="restore-path-selector__empty">请选择待恢复数据</div>
      <div v-else class="restore-path-selector__paths">
        <div v-for="(path, index) in previewPaths" :key="`${sourcePaths[index]}-${index}`" class="restore-path-selector__path-row">
          <code>{{ sourcePaths[index] }}</code>
          <span aria-hidden="true">→</span>
          <code class="restore-path-selector__result">{{ path }}</code>
        </div>
      </div>
      <div v-if="selectedTargetPath && sourcePaths.length" class="restore-path-selector__note">Restic 按所选路径恢复；选择目录时会包含其下内容，并在目标目录中保留原路径层级。</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { buildRestorePathPreview, normalizeRestorePath } from "./preview";

defineOptions({ name: "RestorePathSelector" });

const props = withDefaults(
  defineProps<{
    modelValue?: string[];
    sourcePaths?: string[];
    instanceId?: string | number;
    accessId?: string | number;
    action?: string;
    modalTitle?: string;
    placeholder?: string;
  }>(),
  {
    modelValue: () => [],
    sourcePaths: () => [],
    action: "onListPaths",
    modalTitle: "选择恢复目录",
    placeholder: "例如 /var/lib/app"
  }
);
const emit = defineEmits<{ (event: "update:modelValue", value: string[]): void }>();
const selectedTargetPath = computed(() => props.modelValue[0] || "");
const previewPaths = computed(() => buildRestorePathPreview(selectedTargetPath.value, props.sourcePaths));

function onTargetPathChange(paths: string[]) {
  const selectedPath = paths[paths.length - 1];
  emit("update:modelValue", selectedPath ? [normalizeRestorePath(selectedPath)] : []);
}
</script>

<style lang="less">
.restore-path-selector {
  display: grid;
  gap: 12px;

  .restore-path-selector__preview {
    padding: 12px;
    border: 1px solid #d9e2ec;
    border-radius: 6px;
    background: #f8fafc;
  }
  .restore-path-selector__preview-heading {
    margin-bottom: 8px;
    color: #263343;
    font-size: 13px;
    font-weight: 600;
  }
  .restore-path-selector__empty,
  .restore-path-selector__note {
    color: #7a8796;
    font-size: 12px;
  }
  .restore-path-selector__paths {
    display: grid;
    gap: 6px;
  }
  .restore-path-selector__path-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto minmax(0, 1.2fr);
    align-items: center;
    gap: 8px;
    color: #7a8796;
  }
  .restore-path-selector__path-row code {
    overflow-wrap: anywhere;
    color: #64748b;
  }
  .restore-path-selector__path-row .restore-path-selector__result {
    color: #1456a0;
    font-weight: 600;
  }
  .restore-path-selector__note {
    margin-top: 8px;
  }
}
</style>
