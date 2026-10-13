import { CreateCrudOptionsRet, dict } from "@fast-crud/fast-crud";
import { Modal } from "ant-design-vue";
import { h } from "vue";
import { getBackupPlanList } from "../api";
import { getBackupRunLog, getBackupRunPage, type BackupLog, type BackupRunLogDetail } from "../api";

const logLevelPattern = /^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}) \[(INFO|WARN|ERROR)\] (.*)$/;

function formatSummaryValue(key: string, value: unknown) {
  const labels: Record<string, string> = {
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
  const label = labels[key] || key;
  let renderedValue = String(value);
  if (typeof value === "object") {
    renderedValue = JSON.stringify(value);
  }
  return `${label}: ${renderedValue}`;
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
  const openResticLog = async (row: BackupLog) => {
    const response = (await getBackupRunLog(row.id)) as BackupRunLogDetail;
    const content = response.content || "暂无备份执行日志";
    const summary = response.summary || {};
    const summaryEntries = Object.entries(summary);
    Modal.info({
      title: `备份日志 - ${row.planName || `运行记录 ${row.id}`}`,
      width: 980,
      wrapClassName: "backup-log-modal",
      content: () => {
        let summaryContent = h("span", { class: "backup-log-summary-empty" }, "暂无 summary");
        if (summaryEntries.length > 0) {
          summaryContent = h(
            "div",
            { class: "backup-log-summary-grid" },
            summaryEntries.map(([key, value]) => h("span", { key }, formatSummaryValue(key, value)))
          );
        }
        return h("div", { class: "backup-log-modal-content" }, [
          h("div", { class: "backup-log-summary" }, [h("div", { class: "backup-log-summary-title" }, "Restic Summary"), summaryContent]),
          h("div", { class: "backup-log-terminal" }, renderLogContent(content))
        ]);
      }
    });
  };
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
        width: 90,
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
        status: {
          title: "状态",
          type: "dict-select",
          dict: dict({
            data: [
              { value: "queued", label: "排队中", color: "default" },
              { value: "running", label: "执行中", color: "processing" },
              { value: "success", label: "成功", color: "success" },
              { value: "failed", label: "失败", color: "error" }
            ]
          })
        },
        progress: { title: "进度", type: "number", column: { width: 150, component: { name: "a-progress", vModel: "percent", size: "small" } } },
        stage: { title: "阶段", type: "text", column: { width: 100 } },
        startedAt: { title: "开始时间", type: "datetime", column: { width: 155 } },
        finishedAt: { title: "结束时间", type: "datetime", column: { width: 155 } },
        error: {
          title: "错误信息",
          type: "text",
          column: {
            width: 240,
            cellRender: ({ value }: { value?: string }) => {
              const text = value || "-";
              return (
                <a-tooltip title={value || undefined}>
                  <div style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{text}</div>
                </a-tooltip>
              );
            }
          }
        }
      } as any
    }
  };
}
