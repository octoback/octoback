<template>
  <fs-page class="plan-flow-page">
    <template #header>
      <div class="title">
        备份计划
        <span class="sub">编排来源、计划与存储仓库之间的数据流</span>
      </div>
      <div class="more">
        <a-button type="primary" class="flow-edit-toggle" @click="editMode = !editMode">{{ editMode ? "完成" : "编辑" }}</a-button>
      </div>
    </template>
    <div class="canvas-shell">
      <div class="embedded-crud"><PluginCrud ref="sourceCrud" plugin-type="source" /><PluginCrud ref="repositoryCrud" plugin-type="repository" /></div>
      <div class="zone-bar">
        <div>
          <b>来源</b><a-button v-if="editMode" type="text" title="新增来源" @click="openPluginAdd('source')"><Plus :size="14" /></a-button>
        </div>
        <div>
          <b>计划</b><a-button v-if="editMode" type="text" title="新增计划" @click="openPlanAdd"><Plus :size="14" /></a-button>
        </div>
        <div>
          <b>仓库</b><a-button v-if="editMode" type="text" title="新增仓库" @click="openPluginAdd('repository')"><Plus :size="14" /></a-button>
        </div>
      </div>
      <VueFlow v-model:nodes="nodes" v-model:edges="edges" :node-types="nodeTypes" class="flow-canvas">
        <Background pattern-color="#dbe4f0" :gap="24" /><Controls position="bottom-left" /><MiniMap position="bottom-right" />
      </VueFlow>
    </div>
  </fs-page>
</template>
<script setup lang="ts">
defineOptions({
  name: "BackupPlanFlow"
});

import { nextTick, ref } from "vue";
import { VueFlow, useVueFlow, type Edge, type Node } from "@vue-flow/core";
import { useFs } from "@fast-crud/fast-crud";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import { MiniMap } from "@vue-flow/minimap";
import PlanFlowNode from "./PlanFlowNode.vue";
import "@vue-flow/core/dist/style.css";
import "@vue-flow/core/dist/theme-default.css";
import { getBackupPlanPage, getBackupRunInfo, type BackupPlan, runBackupPlan } from "../api";
import { getPluginInstancePage, type PluginInstanceSimple } from "../../plugin/plugin-api";
import { getCronNextTimes } from "/src/components/cron-editor/utils";
import { Modal } from "ant-design-vue";
import { Plus } from "lucide-vue-next";
import createPlanCrudOptions from "../crud";
import PluginCrud from "../../plugin/index.vue";
import type { PluginInstance } from "../../plugin/plugin-api";
const plans = ref<BackupPlan[]>([]);
const sources = ref<PluginInstanceSimple[]>([]);
const repositories = ref<PluginInstanceSimple[]>([]);
const nodes = ref<Node[]>([]);
const edges = ref<Edge[]>([]);
const { setCenter } = useVueFlow();
const nodeTypes = { flow: PlanFlowNode };
const runStates = ref<Record<number, { status: string; progress: number; stage: string; error?: string }>>({});
const editMode = ref(false);
const { crudExpose: planCrudExpose } = useFs({ createCrudOptions: createPlanCrudOptions });
const sourceCrud = ref<any>();
const repositoryCrud = ref<any>();
function openPluginAdd(kind: "source" | "repository") {
  const crud = kind === "source" ? sourceCrud.value : repositoryCrud.value;
  void (crud?.openAdd() ?? Promise.resolve()).then(() => loadFlow());
}
function openPluginEdit(kind: "source" | "repository", id: number | string) {
  const list = kind === "source" ? sources.value : repositories.value;
  const row = list.find((item) => Number(item.id) === Number(id)) as PluginInstance | undefined;
  if (row) {
    const crud = kind === "source" ? sourceCrud.value : repositoryCrud.value;
    void (crud?.openEdit(row) ?? Promise.resolve()).then(() => loadFlow());
  }
}
function removePlugin(kind: "source" | "repository", id: number | string) {
  const list = kind === "source" ? sources.value : repositories.value;
  const row = list.find((item) => Number(item.id) === Number(id)) as PluginInstance | undefined;
  if (row) {
    const crud = kind === "source" ? sourceCrud.value : repositoryCrud.value;
    void (crud?.remove(row) ?? Promise.resolve()).then(() => loadFlow());
  }
}
function openPlanAdd() {
  void planCrudExpose.openAdd({ row: {} }).then(() => loadFlow());
}
function openPlanEdit(id: number) {
  const plan = plans.value.find((item) => item.id === id);
  if (plan) void planCrudExpose.openEdit({ row: plan }).then(() => loadFlow());
}
function removePlan(id: number) {
  const plan = plans.value.find((item) => item.id === id);
  if (plan) void planCrudExpose.doRemove({ row: plan }).then(() => loadFlow());
}
async function runPlan(planId: number) {
  const plan = plans.value.find((item) => item.id === planId);
  Modal.confirm({
    title: "确认执行备份计划",
    content: `确定要立即执行${plan ? `“${plan.name}”` : "该备份计划"}吗？`,
    okText: "执行",
    cancelText: "取消",
    onOk: async () => {
      const runResult = (await runBackupPlan(planId)) as { logId: number };
      void pollBackupRun(planId, runResult.logId);
    }
  });
}

async function pollBackupRun(planId: number, logId: number) {
  while (true) {
    const runInfo = (await getBackupRunInfo(logId)) as { status: string; progress: number; stage: string; error?: string };
    runStates.value = { ...runStates.value, [planId]: runInfo };
    makeGraph();
    if (runInfo.status !== "queued" && runInfo.status !== "running") {
      if (runInfo.status === "failed") {
        Modal.error({ title: "备份失败", content: runInfo.error || runInfo.stage });
      }
      await loadFlow();
      delete runStates.value[planId];
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
}

function makeGraph() {
  const graphNodes: Node[] = [];
  const graphEdges: Edge[] = [];
  const centerPositionY = 330;
  const rowSpacing = 100;
  const getRowPositionY = (rowIndex: number, rowCount: number) => centerPositionY - ((rowCount - 1) * rowSpacing) / 2 + rowIndex * rowSpacing;
  const repositoryGap = 18;
  const repositoryHeights = repositories.value.map((repository) => {
    const pathCount = plans.value.filter((plan) => plan.repositoryId === Number(repository.id)).length;
    return 50 + (5 + pathCount * 18);
  });
  const repositoryTotalHeight = repositoryHeights.reduce((totalHeight, nodeHeight) => totalHeight + nodeHeight, 0) + Math.max(0, repositories.value.length - 1) * repositoryGap;
  let repositoryPositionY = centerPositionY - repositoryTotalHeight / 2;
  const getPlanStatus = (plan: BackupPlan) => {
    const runState = runStates.value[plan.id];
    if (runState && (runState.status === "running" || runState.status === "queued")) {
      return `${runState.stage || "排队中"} ${runState.progress || 0}%`;
    }
    if (plan.lastStatus === "success") {
      return "运行成功";
    }
    if (plan.lastStatus === "failed") {
      return "运行失败";
    }
    return "未运行";
  };
  sources.value.forEach((source) => {
    const sourceIndex = sources.value.indexOf(source);
    graphNodes.push({
      id: `source-${source.id}`,
      type: "flow",
      position: { x: 48, y: getRowPositionY(sourceIndex, sources.value.length) },
      data: { title: source.name, meta: source.pluginName, kind: "source", editMode: editMode.value, onEdit: () => openPluginEdit("source", source.id), onDelete: () => removePlugin("source", source.id) },
      class: "flow-source"
    });
  });
  repositories.value.forEach((repository) => {
    const repositoryIndex = repositories.value.indexOf(repository);
    const repositoryHeight = repositoryHeights[repositoryIndex];
    const repositoryNodePositionY = repositoryPositionY;
    repositoryPositionY += repositoryHeight + repositoryGap;
    graphNodes.push({
      id: `repository-${repository.id}`,
      type: "flow",
      position: { x: 790, y: repositoryNodePositionY },
      data: {
        title: repository.name,
        meta: repository.pluginName,
        kind: "repository",
        editMode: editMode.value,
        onEdit: () => openPluginEdit("repository", repository.id),
        onDelete: () => removePlugin("repository", repository.id),
        onDetail: () => {
          window.location.hash = `#/sys/repository/${repository.id}`;
        },
        subPaths: plans.value.filter((plan) => plan.repositoryId === Number(repository.id)).map((plan) => ({ id: plan.id, path: plan.repoTag || "未设置标签" }))
      },
      class: "flow-repository"
    });
  });
  plans.value.forEach((plan, planIndex) => {
    const planPositionY = getRowPositionY(planIndex, plans.value.length);
    const sourceNodeId = `source-${plan.sourceId}`;
    const planNodeId = `plan-${plan.id}`;
    const repositoryNodeId = `repository-${plan.repositoryId}`;
    let planClass = "flow-plan";
    let edgeClass = "edge-paused";
    if (plan.enabled) {
      planClass = "flow-plan active";
      edgeClass = "edge-active";
    }
    graphNodes.push({
      id: planNodeId,
      type: "flow",
      position: { x: 410, y: planPositionY },
      data: {
        title: plan.name,
        meta: "下次执行：" + (getCronNextTimes(plan.schedule, 1)[0] || "无"),
        cron: plan.schedule,
        status: getPlanStatus(plan),
        kind: "plan",
        onRun: () => runPlan(plan.id),
        editMode: editMode.value,
        onEdit: () => openPlanEdit(plan.id),
        onDelete: () => removePlan(plan.id)
      },
      class: planClass
    });
    graphEdges.push(
      { id: `source-plan-${plan.id}`, source: sourceNodeId, sourceHandle: "right", target: planNodeId, targetHandle: "left", type: "bezier", animated: plan.enabled, class: edgeClass, markerEnd: "arrowclosed" },
      {
        id: `plan-repository-${plan.id}`,
        source: planNodeId,
        sourceHandle: "right",
        target: repositoryNodeId,
        targetHandle: `path-${plan.id}`,
        type: "bezier",
        animated: plan.enabled,
        class: edgeClass,
        markerEnd: "arrowclosed"
      }
    );
  });
  nodes.value = graphNodes.filter((node, nodeIndex, allNodes) => allNodes.findIndex((candidateNode) => candidateNode.id === node.id) === nodeIndex);
  edges.value = graphEdges;
  nextTick(() => {
    requestAnimationFrame(() => {
      const nodeWidth = 188;
      const nodeHeight = 50;
      const minimumX = Math.min(...nodes.value.map((node) => node.position.x));
      const maximumX = Math.max(...nodes.value.map((node) => node.position.x + nodeWidth));
      const minimumY = Math.min(...nodes.value.map((node) => node.position.y));
      const maximumY = Math.max(...nodes.value.map((node) => node.position.y + nodeHeight));
      const graphCenterX = (minimumX + maximumX) / 2;
      const graphCenterY = (minimumY + maximumY) / 2;
      const zoom = 1.1;
      const canvasElement = document.querySelector<HTMLElement>(".plan-flow-page .flow-canvas");
      const canvasHeight = canvasElement?.clientHeight || 0;
      const topOffset = 50;
      const targetCenterY = minimumY + (canvasHeight / 2 - topOffset) / zoom;
      setCenter(graphCenterX, targetCenterY, { zoom });
    });
  });
}

async function loadFlow() {
  const [planPage, sourcePage, repositoryPage] = await Promise.all([
    getBackupPlanPage({ offset: 0, limit: 1000 }),
    getPluginInstancePage({ pluginType: "source", offset: 0, limit: 1000 }),
    getPluginInstancePage({ pluginType: "repository", offset: 0, limit: 1000 })
  ]);
  plans.value = planPage.records || [];
  sources.value = sourcePage.records || [];
  repositories.value = repositoryPage.records || [];
  makeGraph();
}
loadFlow();
</script>
<style lang="less">
.plan-flow-page {
  .title {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .more {
    margin-left: auto;
    pointer-events: auto;
  }
  .embedded-crud {
    display: none;
  }
  .canvas-shell {
    height: 100%;
    border: 1px solid #dce5f0;
    border-radius: 0;
    background: #fff;
    display: flex;
    flex-direction: column;
    position: relative;
    overflow: hidden;
    .flow-canvas {
      min-height: 0;
      flex: 1;
      background: #fff;
      position: relative;
      z-index: 1;
    }
  }
  .canvas-shell::before {
    content: "";
    position: absolute;
    inset: 52px 0 0;
    pointer-events: none;
    background: linear-gradient(90deg, rgba(59, 130, 246, 0.025) 0 33.33%, rgba(139, 92, 246, 0.025) 33.33% 66.66%, rgba(16, 185, 129, 0.025) 66.66% 100%);
    z-index: 0;
  }
  .zone-bar {
    z-index: 4;
    padding: 10px;
    display: flex;
    justify-content: space-evenly;
    pointer-events: none;
    color: #364152;
    align-items: center;
    background: rgba(255, 255, 255, 0.94);
    border-bottom: 1px solid rgba(220, 229, 240, 0.78);
    box-shadow: 0 1px 0 rgba(255, 255, 255, 0.8);
  }
  .zone-bar > div {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 15px;
    min-height: 28px;
    padding: 0 14px;
    border-radius: 8px;
  }
  .zone-bar b {
    font-weight: 650;
    letter-spacing: 0.02em;
  }
  .zone-bar .ant-btn {
    pointer-events: auto;
    width: 22px;
    height: 22px;
    padding: 0;
    border-radius: 6px;
    color: #64748b;
    background: rgba(100, 116, 139, 0.1);
  }
  .zone-bar .ant-btn:hover {
    color: #fff;
    background: #2563eb;
  }
  .vue-flow__edge-path {
    stroke-width: 2;
    stroke-linecap: round;
  }
  .edge-active .vue-flow__edge-path {
    stroke: #8b5cf6;
    stroke-dasharray: 8 6;
    animation: dash 1s linear infinite;
  }
  .edge-paused .vue-flow__edge-path {
    stroke: #b7c0cd;
    stroke-dasharray: 4 7;
  }
  .vue-flow__controls {
    border: 1px solid #dce5f0;
    box-shadow: 0 5px 15px rgba(34, 55, 84, 0.1);
    border-radius: 10px;
    overflow: hidden;
  }
  @keyframes dash {
    to {
      stroke-dashoffset: -28;
    }
  }
}
</style>
