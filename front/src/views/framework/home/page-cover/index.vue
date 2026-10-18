<template>
  <section class="dashboard">
    <div class="dashboard-header">
      <div>
        <h1>备份概览</h1>
        <p>查看当前 octo-backup 实例的资源配置情况。</p>
      </div>
      <a-button type="primary" @click="go('/sys/repository')"><fs-icon icon="lucide:plus" /> 添加存储仓库</a-button>
    </div>

    <div class="stat-grid">
      <a-card v-for="stat in stats" :key="stat.label" :bordered="true" class="stat-card">
        <div class="stat-label">
          <span>{{ stat.label }}</span
          ><fs-icon :icon="stat.icon" />
        </div>
        <a-statistic :value="stat.value" :loading="loading && stat.loading" />
        <div class="stat-note">{{ stat.note }}</div>
      </a-card>
    </div>

    <div class="content-grid">
      <a-card title="系统状态" class="status-card">
        <template #extra><a-tag color="default">当前实例</a-tag></template>
        <div class="status-empty">
          <div class="status-icon"><fs-icon icon="lucide:activity" /></div>
          <strong>{{ repositoryCount > 0 ? "已配置存储仓库" : "等待配置存储仓库" }}</strong>
          <span>{{ repositoryCount > 0 ? `当前有 ${repositoryCount} 个仓库可用。` : "配置仓库后，备份任务和快照统计会显示在这里。" }}</span>
          <a-button v-if="repositoryCount === 0" type="link" @click="go('/sys/repository')">前往配置</a-button>
        </div>
      </a-card>
      <a-card title="快速入口" class="quick-card">
        <a-list :data-source="quickLinks" :split="true">
          <template #renderItem="{ item }"
            ><a-list-item class="quick-item" @click="go(item.path)"
              ><template #prefix
                ><span class="quick-icon"><fs-icon :icon="item.icon" /></span></template
              ><a-list-item-meta :title="item.title" :description="item.description" /><fs-icon icon="lucide:chevron-right" class="quick-arrow" /></a-list-item
          ></template>
        </a-list>
      </a-card>
    </div>

    <a-card title="数据说明" class="notice-card"
      ><template #extra><fs-icon icon="lucide:info" /></template>
      <p>数据来自当前实例的存储仓库、备份来源、备份计划和 Restic 快照记录。</p></a-card
    >
  </section>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { getPluginInstancePage, getPluginInstanceSnapshots } from "/@/views/sys/plugin/plugin-api";
import { getBackupPlanPage } from "/@/views/sys/plan/api";

const router = useRouter();
const loading = ref(true);
const repositoryCount = ref(0);
const sourceCount = ref(0);
const backupPlanCount = ref(0);
const snapshotCount = ref(0);
const stats = computed(() => [
  { label: "存储仓库", value: repositoryCount.value, note: repositoryCount.value ? "已配置" : "尚未配置", icon: "lucide:database", loading: true },
  { label: "备份主机", value: sourceCount.value, note: sourceCount.value ? "已接入" : "尚未配置", icon: "lucide:server", loading: true },
  { label: "备份任务", value: backupPlanCount.value, note: backupPlanCount.value ? "已配置" : "尚未配置", icon: "lucide:calendar-check", loading: true },
  { label: "恢复快照", value: snapshotCount.value, note: snapshotCount.value ? "可恢复" : "尚无快照", icon: "lucide:history", loading: true }
]);
const quickLinks = [
  { title: "存储仓库", description: "管理 Local、SFTP、S3 和 MinIO 仓库", icon: "lucide:database", path: "/sys/repository" },
  { title: "项目介绍", description: "了解 octo-backup 的定位和能力", icon: "lucide:circle-help", path: "/about/index" }
];
function go(path: string) {
  router.push(path);
}

onMounted(async () => {
  const [repositories, sources, plans] = await Promise.allSettled([
    getPluginInstancePage({ pluginType: "repository", limit: 1000 }),
    getPluginInstancePage({ pluginType: "source", limit: 1 }),
    getBackupPlanPage({ limit: 1 })
  ]);
  if (repositories.status === "fulfilled") {
    repositoryCount.value = Number(repositories.value?.total || 0);
    const snapshotResults = await Promise.allSettled((repositories.value?.records || []).map((record: { id: string | number }) => getPluginInstanceSnapshots(record.id)));
    snapshotCount.value = snapshotResults.reduce((total, result) => {
      if (result.status !== "fulfilled" || !Array.isArray(result.value)) {
        return total;
      }
      return total + result.value.length;
    }, 0);
  }
  if (sources.status === "fulfilled") {
    sourceCount.value = Number(sources.value?.total || 0);
  }
  if (plans.status === "fulfilled") {
    backupPlanCount.value = Number(plans.value?.total || 0);
  }
  loading.value = false;
});
</script>

<style lang="less" scoped>
.dashboard {
  width: 100%;
  box-sizing: border-box;
  min-height: 100%;
  padding: 24px;
  background: #f5f5f5;
  color: rgba(0, 0, 0, 0.85);
}
.dashboard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 0 0 18px;
}
.dashboard-header h1 {
  margin: 0 0 6px;
  font-size: 22px;
  font-weight: 500;
}
.dashboard-header p {
  margin: 0;
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
}
.dashboard-header :deep(.ant-btn) {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin: 0 0 16px;
}
.stat-card :deep(.ant-card-body) {
  padding: 18px 20px 16px;
}
.stat-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
}
.stat-label :deep(.fs-icon),
.stat-label :deep(svg) {
  color: #8c8c8c;
  font-size: 17px;
}
.stat-card :deep(.ant-statistic) {
  margin-top: 15px;
}
.stat-card :deep(.ant-statistic-content) {
  color: rgba(0, 0, 0, 0.85);
  font-size: 28px;
  line-height: 1.15;
}
.stat-note {
  margin-top: 10px;
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}
.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin: 0 0 16px;
}
.content-grid :deep(.ant-card-head-title) {
  font-size: 16px;
  font-weight: 500;
}
.status-card,
.quick-card,
.notice-card {
  border-color: #e8e8e8;
}
.status-empty {
  display: flex;
  min-height: 198px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}
.status-icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  margin-bottom: 12px;
  border-radius: 50%;
  color: #8c8c8c;
  background: #f5f5f5;
  font-size: 20px;
}
.status-empty strong {
  margin-bottom: 7px;
  font-size: 15px;
  font-weight: 500;
}
.status-empty span {
  max-width: 310px;
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
  line-height: 1.6;
}
.status-empty :deep(.ant-btn) {
  padding: 0;
  margin-top: 10px;
}
.quick-card :deep(.ant-card-body) {
  padding: 0 20px;
}
.quick-item {
  cursor: pointer;
  padding: 16px 0;
}
.quick-item:hover .quick-arrow {
  color: #1890ff;
}
.quick-icon {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  margin-right: 12px;
  border-radius: 6px;
  color: #1890ff;
  background: #e6f7ff;
}
.quick-item :deep(.ant-list-item-meta-title) {
  margin-bottom: 3px;
  font-size: 14px;
  font-weight: 500;
}
.quick-item :deep(.ant-list-item-meta-description) {
  font-size: 12px;
}
.quick-arrow {
  color: #bfbfbf;
}
.notice-card {
  margin: 0;
}
.notice-card :deep(.ant-card-body) {
  padding: 14px 20px;
}
.notice-card p {
  margin: 0;
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
  line-height: 1.7;
}
@media (max-width: 900px) {
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .content-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 600px) {
  .dashboard {
    padding: 16px;
  }
  .dashboard-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }
  .stat-grid {
    gap: 10px;
  }
  .stat-card :deep(.ant-card-body) {
    padding: 14px;
  }
}
</style>
