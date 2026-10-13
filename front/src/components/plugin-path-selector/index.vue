<template>
  <div class="plugin-path-selector component-plugin-path-selector">
    <div class="plugin-path-selector__actions">
      <a-button :disabled="!instanceId" @click="openPicker">选择目录</a-button>
      <span v-if="!instanceId" class="plugin-path-selector__hint">保存来源后可浏览 SSH 目录</span>
    </div>
    <a-textarea :value="formatPathInput(modelValue)" :placeholder="placeholder" :auto-size="{ minRows: 3, maxRows: 8 }" @update:value="onInput" />
    <a-modal v-model:open="pickerOpen" destroy-on-close wrap-class-name="plugin-path-selector__modal" :title="modalTitle" :confirm-loading="loading" @ok="confirmSelection">
      <a-alert v-if="errorMessage" type="error" :message="errorMessage" show-icon />
      <div v-if="selectedPaths.length" class="plugin-path-selector__selected">
        <span class="plugin-path-selector__selected-label">已选择：</span>
        <a-tag v-for="path in visibleSelectedPaths" :key="path" type="primary" color="blue" closable @close="removeSelectedPath(path)">{{ path }}</a-tag>
        <a-tag v-if="selectedPaths.length > visibleSelectedPaths.length" type="primary" color="blue">+{{ selectedPaths.length - visibleSelectedPaths.length }}</a-tag>
      </div>
      <a-spin :spinning="loading">
        <a-tree v-model:checked-keys="checkedKeys" checkable :check-strictly="true" block-node :tree-data="treeData" :field-names="treeFieldNames" :load-data="loadData">
          <template #title="node">
            <span class="plugin-path-selector__tree-title">
              <span class="plugin-path-selector__tree-path">{{ node.title }}</span>
              <span v-if="hasNodeCounts(node)" class="plugin-path-selector__tree-count">共 {{ node.fileCount + node.directoryCount }} 项</span>
            </span>
          </template>
        </a-tree>
      </a-spin>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
defineOptions({
  name: "PluginPathSelector"
});

import { computed, ref } from "vue";
import { executePluginInstanceAction } from "/src/views/sys/plugin/plugin-api";
import { formatPathInput, getDirectoryRequestPath, normalizePathNodes, parsePathInput, updatePathNodeChildren, updatePathNodeCounts, type PluginPathNode } from "./helpers";

const props = withDefaults(defineProps<{ modelValue?: string | string[]; placeholder?: string; modalTitle?: string; instanceId?: string | number; action?: string; accessId?: string | number }>(), {
  placeholder: "选择目录或粘贴路径，每行一个",
  modalTitle: "选择备份目录",
  action: "onListPaths"
});
const emit = defineEmits<{ (event: "update:modelValue", value: string[]): void }>();
const pickerOpen = ref(false);
const loading = ref(false);
const errorMessage = ref("");
const treeData = ref<PluginPathNode[]>([]);
const checkedKeys = ref<string[] | { checked: string[]; halfChecked: string[] }>([]);
const treeFieldNames = { title: "title", key: "key", children: "children" };
const selectedPaths = computed(() => (Array.isArray(checkedKeys.value) ? checkedKeys.value : checkedKeys.value.checked));
const visibleSelectedPaths = computed(() => selectedPaths.value.slice(0, 3));

function onInput(value: string) {
  emit("update:modelValue", parsePathInput(value));
}
async function openPicker() {
  if (!props.instanceId) return;
  pickerOpen.value = true;
  loading.value = true;
  errorMessage.value = "";
  treeData.value = [];
  try {
    const result = await loadPaths(getDirectoryRequestPath(undefined, undefined));
    const rootNode: PluginPathNode = { title: "/", key: "/", isLeaf: false, children: normalizePathNodes(result) };
    treeData.value = [rootNode];
    updatePathNodeCounts(treeData.value, "/", result);
    checkedKeys.value = parsePathInput(props.modelValue);
  } catch (error: any) {
    errorMessage.value = error?.message || "读取 SSH 目录失败";
  } finally {
    loading.value = false;
  }
}
async function loadPaths(path: string) {
  return executePluginInstanceAction(props.instanceId!, props.action, { accessId: props.accessId, path: getDirectoryRequestPath(path, undefined) });
}
async function loadData(node: PluginPathNode & { children?: PluginPathNode[] }) {
  if (node.children) {
    return;
  }
  try {
    const result = await loadPaths(node.key);
    const children = normalizePathNodes(result);
    updatePathNodeCounts(treeData.value, node.key, result);
    updatePathNodeChildren(treeData.value, node.key, children);
    if (!children.length) {
      node.isLeaf = true;
    }
  } catch (error: any) {
    errorMessage.value = error?.message || `读取目录 ${node.key} 失败`;
    throw error;
  }
}
function confirmSelection() {
  const selectedKeys = Array.isArray(checkedKeys.value) ? checkedKeys.value : checkedKeys.value.checked;
  emit("update:modelValue", parsePathInput(selectedKeys));
  pickerOpen.value = false;
}
function removeSelectedPath(path: string) {
  const remainingPaths = selectedPaths.value.filter((selectedPath) => selectedPath !== path);
  checkedKeys.value = remainingPaths;
}
function hasNodeCounts(node: PluginPathNode) {
  return Number.isFinite(node.fileCount) && Number.isFinite(node.directoryCount);
}
</script>

<style lang="less">
.plugin-path-selector {
  display: grid;
  gap: 8px;
  .plugin-path-selector__actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .plugin-path-selector__hint {
    color: #8c8c8c;
    font-size: 12px;
  }
  .plugin-path-selector__selected {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 4px;
    margin-bottom: 8px;
  }
  .plugin-path-selector__selected-label {
    color: #666;
    font-size: 12px;
  }
}

.plugin-path-selector__modal {
  .plugin-path-selector__tree-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    width: 100%;
    min-width: 0;
  }
  .plugin-path-selector__tree-path {
    overflow: hidden;
    color: #262626;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .plugin-path-selector__tree-count {
    flex: 0 0 auto;
    min-width: 72px;
    padding-left: 12px;
    border-left: 1px solid #e5e7eb;
    color: #8c8c8c;
    font-size: 12px;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
}
</style>
