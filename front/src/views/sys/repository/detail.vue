<template>
  <fs-page class="repository-detail-page">
    <a-page-header class="repository-detail-header" :title="repositoryTitle" @back="goBack">
      <template #subTitle>
        <span>Restic 仓库快照</span>
      </template>
      <template #extra>
        <a-space>
          <a-tooltip title="校验仓库数据完整性，不会修改仓库内容">
            <a-button type="primary" :loading="actionLoading" @click="checkRepository">检查</a-button>
          </a-tooltip>
          <a-tooltip title="重新查询仓库快照和统计数据，并更新本地缓存">
            <a-button type="primary" :loading="loading" @click="loadSnapshots(true)">刷新</a-button>
          </a-tooltip>
        </a-space>
      </template>
    </a-page-header>
    <a-alert v-if="errorMessage" type="error" show-icon :message="errorMessage" />
    <a-spin :spinning="loading" class="repository-detail-loading">
      <div class="repository-detail-content">
        <section class="snapshot-list-panel">
          <div class="panel-heading">
            <div>
              <span class="panel-kicker">REPOSITORY</span>
              <h2>Snapshots</h2>
            </div>
            <span>
              <a-tag color="blue">{{ snapshots.length }} 个</a-tag>
              <a-tag color="green">占 {{ formatBytes(repositoryStats?.total_size) }}</a-tag>
            </span>
          </div>
          <a-empty v-if="!snapshots.length && !loading" description="仓库中暂无快照" />
          <div v-else class="snapshot-list">
            <div v-for="group in snapshotGroups" :key="group.tag" class="snapshot-group">
              <div class="snapshot-group-title">
                <TagOutlined /> {{ group.tag }} <span>{{ group.items.length }}</span>
              </div>
              <button v-for="snapshot in group.items" :key="snapshot.id" class="snapshot-item" :class="{ active: selectedSnapshot?.id === snapshot.id }" type="button" @click="selectSnapshot(snapshot)">
                <span class="snapshot-status"> <DatabaseOutlined /> </span
                ><span class="snapshot-item-body"
                  ><span class="snapshot-time-row"
                    ><strong>{{ formatDate(snapshot.time) }}</strong
                    ><a-tag v-for="tag in snapshot.tags?.length ? snapshot.tags : ['无标签']" :key="tag" color="blue">{{ tag }}</a-tag></span
                  ><span class="snapshot-id">{{ snapshot.id }}</span></span
                >
                <RightOutlined />
              </button>
            </div>
          </div>
        </section>
        <section class="snapshot-detail-panel">
          <template v-if="selectedSnapshot">
            <div class="detail-heading">
              <div>
                <span class="panel-kicker">SNAPSHOT DETAIL</span>
                <h1>{{ formatDate(selectedSnapshot.time) }}</h1>
              </div>
              <a-space>
                <a-tag color="green">已保存</a-tag>
                <a-button danger :loading="snapshotDeleting" @click="deleteSelectedSnapshot">删除快照</a-button>
              </a-space>
            </div>
            <a-divider />
            <div class="detail-grid">
              <div>
                <span>Snapshot ID</span><strong>{{ selectedSnapshot.id }}</strong>
              </div>
              <div>
                <span>主机</span><strong>{{ selectedSnapshot.hostname || "未知" }}</strong>
              </div>
              <div>
                <span>用户</span><strong>{{ selectedSnapshot.username || "未知" }}</strong>
              </div>
              <div>
                <span>标签</span><strong>{{ formatTags(selectedSnapshot.tags) }}</strong>
              </div>
            </div>
            <div class="summary-section">
              <div class="section-title">备份 Summary</div>
              <div v-if="snapshotSummaryEntries.length" class="summary-grid">
                <div v-for="entry in snapshotSummaryEntries" :key="entry.key" class="summary-item">
                  <span>{{ entry.label }}</span>
                  <strong>{{ entry.value }}</strong>
                </div>
              </div>
              <a-empty v-else :image="null" description="暂无备份 Summary" />
            </div>
            <div class="paths-section">
              <div class="mb-2">
                <a-button type="primary" :disabled="!restorePaths.length" @click="openRestoreDialog">恢复所选</a-button>
              </div>
              <div v-if="selectedSnapshot.paths?.length" class="backup-path-list">
                <div class="paths-heading">
                  <div class="section-title">恢复备份目录</div>
                </div>
                <a-checkbox-group v-model:value="selectedBackupPaths">
                  <label v-for="path in selectedSnapshot.paths" :key="path" class="backup-path-item">
                    <a-checkbox :value="path" />
                    <FolderOutlined />
                    <code>{{ path }}</code>
                  </label>
                </a-checkbox-group>
              </div>
              <a-empty v-else :image="null" description="没有备份路径" />
            </div>
            <div class="browser-section">
              <div class="section-title">恢复指定路径</div>
              <a-spin :spinning="browserLoading">
                <a-directory-tree :key="selectedSnapshot.id" v-model:checked-keys="selectedPaths" :tree-data="browserTree" :load-data="loadBrowserNode" :show-icon="true" checkable block-node />
              </a-spin>
            </div>
          </template>
          <a-empty v-else description="选择一个快照查看详情" />
        </section>
      </div>
    </a-spin>
    <a-modal v-model:open="restoreVisible" title="创建恢复任务" :confirm-loading="restoreLoading" width="800px" @ok="submitRestore">
      <a-alert type="warning" show-icon message="如果目标目录已有同名文件，恢复时会用备份中的文件覆盖它" />
      <a-form class="restore-form mt-2">
        <a-form-item label="目标 SSH 授权" required>
          <AccessSelector v-model="restoreForm.targetAccessId" />
        </a-form-item>
        <a-form-item label="恢复到主机目录" required>
          <RestorePathSelector v-model="restoreForm.targetPath" :source-paths="restorePaths" :instance-id="restoreForm.targetAccessId" action="onListPaths" @path-status="onRestorePathStatus" />
        </a-form-item>
      </a-form>
    </a-modal>
  </fs-page>
</template>
<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { DatabaseOutlined, FileOutlined, FolderOutlined, RightOutlined, TagOutlined } from "@ant-design/icons-vue";
import { message, Modal } from "ant-design-vue";
import {
  checkPluginInstanceRepository,
  getPluginInstanceInfo,
  getPluginInstanceSnapshotBrowser,
  getPluginInstanceSnapshots,
  getPluginInstanceStats,
  refreshPluginInstanceRepositoryData,
  forgetPluginInstanceSnapshot
} from "/@/views/sys/plugin/plugin-api";
import { createRestoreTask } from "/@/views/sys/restore-task/api";
import AccessSelector from "/@/components/access-selector/index.vue";
import RestorePathSelector from "/@/components/restore-path-selector/index.vue";
import { compactRestorePaths } from "/@/components/restore-path-selector/preview";
interface Snapshot {
  id: string;
  time?: string;
  hostname?: string;
  username?: string;
  tags?: string[];
  paths?: string[];
  summary?: Record<string, unknown>;
  sourceId?: number;
  repositoryId?: number;
  snapshotId?: string;
}
interface BrowserEntry {
  path?: string;
  type?: string;
  size?: number;
  mode?: string;
}
const route = useRoute();
const loading = ref(false);
const errorMessage = ref("");
const repositoryName = ref("");
const snapshots = ref<Snapshot[]>([]);
const selectedSnapshot = ref<Snapshot>();
const statsLoading = ref(false);
const repositoryStats = ref<{ total_size?: number; total_file_count?: number; snapshot_count?: number }>();
const repositoryTitle = computed(() => (repositoryName.value ? `${repositoryName.value}` : "仓库详情"));
const browserLoading = ref(false);
const browserEntries = ref<BrowserEntry[]>([]);
const browserLoaded = ref(false);
const actionLoading = ref(false);
const snapshotDeleting = ref(false);
const restoreVisible = ref(false);
const restoreLoading = ref(false);
const selectedPaths = ref<string[]>([]);
const selectedBackupPaths = ref<string[]>([]);
const restorePaths = computed(() => compactRestorePaths([...selectedBackupPaths.value, ...selectedPaths.value]));
const restoreForm = ref({ targetAccessId: 0, targetPath: [] as string[] });
const restorePathIsEmpty = ref<boolean | undefined>();
const snapshotSummaryEntries = computed(() => {
  const summary = selectedSnapshot.value?.summary;
  if (!summary) {
    return [];
  }
  const labels: Record<string, string> = {
    backup_start: "开始时间",
    backup_end: "结束时间",
    total_files_processed: "处理文件",
    total_bytes_processed: "处理字节",
    total_duration: "耗时（秒）",
    files_new: "新增文件",
    files_changed: "变更文件",
    files_unmodified: "未变更文件",
    dirs_new: "新增目录",
    dirs_changed: "变更目录",
    dirs_unmodified: "未变更目录",
    data_added: "新增数据",
    data_added_packed: "新增压缩数据"
  };
  return Object.entries(summary).map(([key, value]) => {
    let renderedValue = String(value);
    if ((key === "backup_start" || key === "backup_end") && typeof value === "string") {
      renderedValue = formatDate(value);
    }
    if ((key === "data_added" || key === "data_added_packed" || key === "total_bytes_processed") && typeof value === "number") {
      renderedValue = formatBytes(value);
    }
    if (typeof value === "object") {
      renderedValue = JSON.stringify(value);
    }
    return { key, label: labels[key] || key, value: renderedValue };
  });
});
const snapshotGroups = computed(() => {
  const groups = new Map<string, Snapshot[]>();
  for (const snapshot of snapshots.value) {
    const tag = formatTags(snapshot.tags);
    const items = groups.get(tag) || [];
    items.push(snapshot);
    groups.set(tag, items);
  }
  return Array.from(groups, ([tag, items]) => ({ tag, items }));
});
const browserTree = computed(() => {
  const root = { key: "/", title: "/", icon: h(FolderOutlined), isLeaf: false };
  if (!browserLoaded.value) {
    return [root];
  }
  const roots: any[] = [];
  const lookup = new Map<string, any>();
  for (const entry of browserEntries.value) {
    const parts = String(entry.path || "/")
      .split("/")
      .filter(Boolean);
    let parent = roots;
    let currentPath = "";
    for (const part of parts) {
      currentPath += `/${part}`;
      let node = lookup.get(currentPath);
      if (!node) {
        node = { key: currentPath, title: part, icon: h(FolderOutlined), children: [], isLeaf: false };
        lookup.set(currentPath, node);
        parent.push(node);
      }
      parent = node.children;
    }
    const node = lookup.get(currentPath);
    if (node && entry.type !== "dir") {
      node.isLeaf = true;
      node.icon = h(FileOutlined);
      delete node.children;
    }
  }
  root.children = roots;
  root.isLeaf = roots.length === 0;
  return [root];
});
function goBack() {
  window.location.hash = "#/sys/repository";
}
function formatDate(value?: string) {
  if (!value) return "未知时间";
  return new Date(value).toLocaleString("zh-CN", { hour12: false });
}
function formatTags(tags?: string[]) {
  return tags?.length ? tags.join(", ") : "无标签";
}
function formatBytes(value?: number) {
  if (!value) return "-";
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`;
  return `${(value / 1024 / 1024).toFixed(2)} MiB`;
}
async function selectSnapshot(snapshot: Snapshot) {
  selectedSnapshot.value = snapshot;
  browserLoading.value = true;
  browserEntries.value = [];
  browserLoaded.value = false;
  selectedPaths.value = [];
  selectedBackupPaths.value = [];
  browserLoading.value = false;
}
async function submitRestore() {
  if (!selectedSnapshot.value || !restoreForm.value.targetAccessId || restoreForm.value.targetPath.length !== 1) {
    message.error("请选择恢复路径、目标主机和目标目录");
    return;
  }
  if (restorePathIsEmpty.value === false) {
    message.error("请选择一个空目录");
    return;
  }
  restoreLoading.value = true;
  try {
    await createRestoreTask({
      repositoryId: Number(route.params.id),
      snapshotId: selectedSnapshot.value.id,
      sourceId: selectedSnapshot.value.sourceId,
      targetAccessId: restoreForm.value.targetAccessId,
      targetPath: restoreForm.value.targetPath[0],
      selectedPaths: restorePaths.value
    });
    message.success("恢复任务已创建");
    restoreVisible.value = false;
  } catch (error: any) {
    message.error(error?.message || "创建恢复任务失败");
  } finally {
    restoreLoading.value = false;
  }
}
function deleteSelectedSnapshot() {
  if (!selectedSnapshot.value) {
    return;
  }
  const snapshot = selectedSnapshot.value;
  Modal.confirm({
    title: "确认删除快照",
    content: `删除快照“${formatDate(snapshot.time)}”后将清理其不再使用的数据，操作不可恢复。确定继续吗？`,
    okText: "删除",
    okType: "danger",
    cancelText: "取消",
    onOk: async () => {
      snapshotDeleting.value = true;
      try {
        await forgetPluginInstanceSnapshot(String(route.params.id), snapshot.id);
        message.success("快照已删除");
        await loadSnapshots();
      } catch (error: any) {
        message.error(error?.message || "删除快照失败");
      } finally {
        snapshotDeleting.value = false;
      }
    }
  });
}
function openRestoreDialog() {
  restoreForm.value = { targetAccessId: 0, targetPath: [] };
  restorePathIsEmpty.value = undefined;
  restoreVisible.value = true;
}
function onRestorePathStatus(isEmpty: boolean | undefined) {
  restorePathIsEmpty.value = isEmpty;
}
async function loadBrowserNode() {
  if (browserLoaded.value || !selectedSnapshot.value) {
    return;
  }
  browserLoading.value = true;
  try {
    const result = await getPluginInstanceSnapshotBrowser(String(route.params.id), selectedSnapshot.value.id);
    browserEntries.value = Array.isArray(result) ? result : [];
    browserLoaded.value = true;
  } finally {
    browserLoading.value = false;
  }
}
async function loadSnapshots(refresh = false) {
  loading.value = true;
  statsLoading.value = true;
  errorMessage.value = "";
  try {
    const infoPromise = getPluginInstanceInfo(String(route.params.id));
    const detailPromise = refresh
      ? refreshPluginInstanceRepositoryData(String(route.params.id))
      : Promise.all([getPluginInstanceSnapshots(String(route.params.id)), getPluginInstanceStats(String(route.params.id))]).then(([result, stats]) => ({ snapshots: result, stats }));
    const [info, detail] = await Promise.all([infoPromise, detailPromise]);
    repositoryName.value = info.name;
    snapshots.value = Array.isArray(detail?.snapshots) ? [...detail.snapshots].sort((left, right) => String(right.time || "").localeCompare(String(left.time || ""))) : [];
    repositoryStats.value = detail?.stats || undefined;
    browserEntries.value = [];
    browserLoaded.value = false;
    if (snapshots.value[0]) {
      await selectSnapshot(snapshots.value[0]);
    }
  } catch (error: any) {
    errorMessage.value = error?.message || "加载仓库快照失败";
  } finally {
    loading.value = false;
    statsLoading.value = false;
  }
}
onMounted(loadSnapshots);
async function checkRepository() {
  actionLoading.value = true;
  try {
    await checkPluginInstanceRepository(String(route.params.id));
    message.success("仓库检查完成");
  } catch (error: any) {
    message.error(error?.message || "仓库检查失败");
  } finally {
    actionLoading.value = false;
  }
}
</script>
<style lang="less">
.repository-detail-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: #f5f7fa;

  .repository-detail-header {
    background: #fff;
    border-bottom: 1px solid #e5e7eb;
  }

  .ant-spin-nested-loading {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;

    .ant-spin-container {
      height: 100%;
    }
  }

  .fs-page-content {
    display: flex;
    flex-direction: column;
    flex: 1;
  }

  .repository-detail-content {
    display: grid;
    grid-template-columns: minmax(280px, 34%) 1fr;
    height: 100%;
    min-height: 0;
    background: #fff;
  }

  .snapshot-list-panel {
    min-height: 0;
    overflow-y: auto;
    border-right: 1px solid #e5e7eb;
    padding: 12px 0;
  }

  .snapshot-detail-panel {
    min-height: 0;
    overflow-y: auto;
    padding: 18px 24px;
  }

  .panel-heading,
  .detail-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
  }

  .detail-heading {
    padding: 0;
  }

  .panel-kicker {
    color: #8b95a5;
    font-size: 11px;
    letter-spacing: 0.12em;
  }

  h1,
  h2 {
    margin: 4px 0 0;
    color: #17212f;
  }

  h1 {
    font-size: 22px;
  }

  h2 {
    font-size: 16px;
  }

  .snapshot-list {
    margin-top: 10px;
  }

  .snapshot-group + .snapshot-group {
    margin-top: 12px;
  }

  .snapshot-group-title {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 16px;
    color: #64748b;
    font-size: 12px;
    font-weight: 600;
  }

  .snapshot-group-title span {
    color: #a0aaba;
    font-weight: 400;
  }

  .snapshot-item {
    display: flex;
    align-items: center;
    width: 100%;
    gap: 8px;
    padding: 10px 16px;
    border: 0;
    border-left: 3px solid transparent;
    background: transparent;
    color: #4b5563;
    text-align: left;
    cursor: pointer;
  }

  .snapshot-item:hover,
  .snapshot-item.active {
    background: #eef6ff;
    border-left-color: #1677ff;
  }

  .snapshot-status {
    color: #1677ff;
    font-size: 17px;
  }

  .snapshot-item-body {
    display: grid;
    flex: 1;
    gap: 3px;
    min-width: 0;
  }

  .snapshot-item-body strong {
    color: #17212f;
  }

  .snapshot-time-row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }

  .snapshot-time-row :deep(.ant-tag) {
    margin: 0;
  }

  .snapshot-item-body span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .snapshot-id {
    color: #9aa4b2;
    font-family: monospace;
    font-size: 12px;
  }

  .snapshot-tag {
    color: #64748b;
    font-size: 12px;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 14px 18px;
  }

  .detail-grid div {
    display: grid;
    gap: 4px;
  }

  .detail-grid span {
    color: #8b95a5;
    font-size: 13px;
  }

  .detail-grid strong {
    color: #263343;
    font-size: 14px;
    word-break: break-word;
  }

  .paths-section,
  .browser-section,
  .summary-section {
    margin-top: 18px;
  }

  .summary-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    border-top: 1px solid #e5e7eb;
  }

  .summary-item {
    display: grid;
    align-content: start;
    gap: 4px;
    min-width: 0;
    padding: 8px 10px;
    border-bottom: 1px solid #e5e7eb;
  }

  .summary-item span {
    color: #8b95a5;
    font-size: 12px;
  }

  .summary-item strong {
    color: #263343;
    font-size: 13px;
    overflow-wrap: anywhere;
  }

  .paths-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .backup-path-list {
    padding: 6px 10px;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
  }

  .backup-path-list :deep(.ant-checkbox-group) {
    display: grid;
    width: 100%;
    gap: 4px;
  }

  .backup-path-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-width: 0;
    color: #64748b;
  }

  .backup-path-item code {
    min-width: 0;
    color: #263343;
    font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
    overflow-wrap: anywhere;
  }

  .browser-section {
    padding: 12px;
    border: 1px solid #e5e7eb;
    border-radius: 6px;

    .section-title {
      margin-bottom: 12px;
    }

    .ant-tree .ant-tree-switcher {
      display: flex;
      align-items: center;
      justify-content: center;
    }
  }

  .browser-section :deep(.ant-tree-node-content-wrapper),
  .browser-section :deep(.ant-tree-iconEle) {
    display: inline-flex;
    align-items: center;
  }

  .browser-section :deep(.ant-tree-iconEle) {
    justify-content: center;
    height: 24px;
  }

  .section-title {
    margin-bottom: 10px;
    color: #263343;
    font-weight: 600;
  }

  .paths-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  @media (max-width: 800px) {
    .repository-detail-content {
      grid-template-columns: 1fr;
      grid-template-rows: minmax(240px, 40%) minmax(0, 1fr);
    }

    .snapshot-list-panel {
      min-height: 0;
      border-right: 0;
      border-bottom: 1px solid #e5e7eb;
    }

    .snapshot-detail-panel {
      min-height: 0;
      padding: 16px;
    }

    .detail-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .summary-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
}
</style>
