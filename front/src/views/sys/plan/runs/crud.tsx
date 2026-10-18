import { CreateCrudOptionsRet, dict } from "@fast-crud/fast-crud";
import { Modal } from "ant-design-vue";
import { h } from "vue";
import { getBackupPlanList } from "../api";
import { getBackupRunLog, getBackupRunPage, type BackupLog, type BackupRunLogDetail } from "../api";

const logLevelPattern = /^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}) \[(INFO|WARN|ERROR)\] (.*)$/;

function formatSummaryDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

const summaryLabels: Record<string, string> = {
  backup_start: "开始时间",
  backup_end: "结束时间",
  snapshot_id: "快照 ID",
  total_files_processed: "处理文件",
  total_bytes_processed: "处理字节",
  total_duration: "耗时（秒）",
  files_new: "新增文件",
  files_changed: "变更文件",
  files_unmodified: "未变更文件",
  dirs_new: "新增目录",
  dirs_changed: "变更目录",
  dirs_unmodified: "未变更目录"
};

function formatSummaryValue(key: string, value: unknown) {
  let renderedValue = String(value);
  if ((key === "backup_start" || key === "backup_end") && typeof value === "string") {
    renderedValue = formatSummaryDate(value);
  }
  if (typeof value === "object") {
    renderedValue = JSON.stringify(value, null, 2);
  }
  return renderedValue;
}

function formatSummaryLabel(key: string) {
  return summaryLabels[key] || key;
}

function getSummaryEntries(summary: Record<string, unknown>) {
  const preferredKeys = [
    "backup_start",
    "backup_end",
    "snapshot_id",
    "total_files_processed",
    "total_bytes_processed",
    "total_duration",
    "files_new",
    "files_changed",
    "files_unmodified",
    "dirs_new",
    "dirs_changed",
    "dirs_unmodified"
  ];
  const entries = Object.entries(summary);
  const preferredEntries = preferredKeys.filter((key) => Object.prototype.hasOwnProperty.call(summary, key)).map((key) => [key, summary[key]] as [string, unknown]);
  const preferredSet = new Set(preferredKeys);
  return preferredEntries.concat(entries.filter(([key]) => !preferredSet.has(key)));
}

function renderSummary(summary: Record<string, unknown>) {
  const entries = getSummaryEntries(summary);
  if (entries.length === 0) {
    return h("span", { class: "backup-log-summary-empty" }, "暂无 summary");
  }
  return h(
    "div",
    { class: "backup-log-summary-grid" },
    entries.map(([key, value]) =>
      h("div", { key, class: ["backup-log-summary-item", key === "snapshot_id" && "backup-log-summary-snapshot"] }, [
        h("span", { class: "backup-log-summary-key" }, formatSummaryLabel(key)),
        h("span", { class: "backup-log-summary-value" }, formatSummaryValue(key, value))
      ])
    )
  );
}

function renderLogContent(content: string) {
  return content.split("\n").map((line, index) => {
    const match = logLevelPattern.exec(line);
    if (!match) {
      return h("div", { key: index, class: "backup-log-line backup-log-line-plain" }, line || " ");
    }
    const [, timestamp, level, message] = match;
    return h("div", { key: index, class: `backup-log-line backup-log-level-${level.toLowerCase()}` }, [
      h("span", { class: "backup-log-time" }, timestamp),
      h("span", { class: "backup-log-level" }, level),
      h("span", { class: "backup-log-message" }, message)
    ]);
  });
}

export default function (): CreateCrudOptionsRet<BackupLog> {
  const openRepository = (row: BackupLog) => {
    if (row.repositoryId) {
      window.location.hash = `#/sys/repository/${row.repositoryId}`;
    }
  };
  const loadLogDetail = async (row: BackupLog) => {
    return (await getBackupRunLog(row.id)) as BackupRunLogDetail;
  };
  const openSummary = async (row: BackupLog) => {
    const response = await loadLogDetail(row);
    const summary = response.summary || {};
    Modal.info({
      title: `备份 Summary - ${row.planName || `运行记录 ${row.id}`}`,
      width: 820,
      wrapClassName: "backup-log-modal",
      content: () => h("div", { class: "backup-log-summary-dialog" }, [h("div", { class: "backup-log-summary-title" }, "Restic Summary"), renderSummary(summary)])
    });
  };
  const openResticLog = async (row: BackupLog) => {
    const response = await loadLogDetail(row);
    const content = response.content || "暂无备份执行日志";
    Modal.info({
      title: `执行日志 - ${row.planName || `运行记录 ${row.id}`}`,
      width: 980,
      wrapClassName: "backup-log-modal",
      content: () => h("div", { class: "backup-log-modal-content" }, [h("div", { class: "backup-log-terminal" }, renderLogContent(content))])
    });
  };
  const statusDict = dict({
    data: [
      { value: "queued", label: "排队中", color: "default" },
      { value: "running", label: "执行中", color: "processing" },
      { value: "success", label: "成功", color: "success" },
      { value: "failed", label: "失败", color: "error" }
    ]
  });
  return {
    crudOptions: {
      request: {
        pageRequest: ({ query, page }: any) => {
          return getBackupRunPage({
            offset: page.offset,
            limit: page.limit,
            planId: query?.planId
          });
        }
      },
      rowHandle: {
        fixed: "right",
        width: 160,
        buttons: {
          view: { show: false },
          copy: { show: false },
          edit: { show: false },
          remove: { show: false },
          repository: {
            show: true,
            text: "",
            title: "查看备份仓库文件",
            type: "link",
            icon: "ion:folder-open-outline",
            click: ({ row }: { row: BackupLog }) => openRepository(row)
          },
          summary: {
            show: true,
            text: "",
            title: "查看备份 Summary",
            type: "link",
            icon: "ion:stats-chart-outline",
            click: ({ row }: { row: BackupLog }) => openSummary(row)
          },
          log: {
            show: true,
            text: "",
            title: "查看 Restic 日志",
            type: "link",
            icon: "ion:document-text-outline",
            click: ({ row }: { row: BackupLog }) => openResticLog(row)
          }
        }
      },
      columns: {
        id: { title: "ID", type: "text", column: { width: 70 } },
        planId: {
          title: "备份计划",
          type: "dict-select",
          search: { show: true },
          dict: dict({
            value: "id",
            label: "name",
            getData: async () => getBackupPlanList()
          }),
          column: { show: false }
        },
        planName: { title: "备份计划", type: "text", column: { width: 150 } },
        progress: { title: "进度", type: "number", column: { width: 150, component: { name: "a-progress", vModel: "percent", size: "small" } } },
        stage: { title: "阶段", type: "text", column: { width: 100 } },
        status: {
          title: "状态",
          type: "dict-select",
          search: { show: true },
          dict: statusDict,
          column: {
            width: 120,
            cellRender: ({ value, row }: any) => {
              return (
                <div class="flex items-center">
                  <fs-values-format modelValue={value} dict={statusDict}></fs-values-format>
                  {value === "failed" && (
                    <a-tooltip class="ml-2" title={row.error}>
                      <fs-icon icon="bx:error-circle" size={16}></fs-icon>
                    </a-tooltip>
                  )}
                </div>
              );
            }
          }
        },
        startedAt: { title: "开始时间", type: "datetime", column: { width: 155 } },
        finishedAt: { title: "结束时间", type: "datetime", column: { width: 155 } },
        error: {
          title: "错误信息",
          type: "text",
          column: {
            show: false
          }
        }
      } as any
    }
  };
}
