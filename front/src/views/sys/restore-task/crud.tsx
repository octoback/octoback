import { CreateCrudOptionsRet, DelReq, dict } from "@fast-crud/fast-crud";
import { Modal } from "ant-design-vue";
import { h } from "vue";
import { deleteRestoreTask, getRestoreTaskLog, getRestoreTaskPage, type RestoreTask } from "./api";

function renderLogContent(logs: Array<{ id: number; level: string; message: string; createdAt: number }>) {
  return logs.map((log) => {
    const level = log.level || "INFO";
    const timestamp = new Date(log.createdAt).toLocaleString("zh-CN", { hour12: false });
    const message = log.message;
    return h("div", { key: log.id, class: `restore-task-log-line restore-task-log-level-${level.toLowerCase()}` }, [
      h("span", { class: "restore-task-log-time" }, timestamp),
      h("span", { class: "restore-task-log-level" }, level),
      h("span", { class: "restore-task-log-message" }, message)
    ]);
  });
}

export default function (): CreateCrudOptionsRet<RestoreTask> {
  const showLog = async (row: RestoreTask) => {
    const result = (await getRestoreTaskLog(row.id)) as { logs?: Array<{ id: number; level: string; message: string; createdAt: number }> };
    Modal.info({
      title: `恢复日志 - 任务 ${row.id}`,
      width: 980,
      wrapClassName: "restore-task-log-modal",
      content: h("div", { class: "restore-task-log-terminal" }, result.logs?.length ? renderLogContent(result.logs) : "暂无恢复执行日志")
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
      request: { pageRequest: ({ query, page }: any) => getRestoreTaskPage({ offset: page.offset, limit: page.limit, status: query?.status }), delRequest: ({ row }: DelReq<RestoreTask>) => deleteRestoreTask(row.id) },
      columns: {
        id: { title: "Id", type: "text", column: { width: 70 } },
        title: { title: "恢复标题", type: "text", column: { width: 220 } },
        snapshotId: {
          title: "快照 Id",
          type: "text",
          column: {
            width: 180,
            cellRender: ({ value }: { value: string }) => h("a-typography-text", { ellipsis: { tooltip: value }, copyable: { text: value }, class: "restore-task-snapshot-id" }, () => value)
          }
        },
        repositoryName: { title: "存储仓库", type: "text", column: { width: 150 } },
        targetAccessName: { title: "目标主机", type: "text", column: { width: 150 } },
        targetPath: { title: "目标目录", type: "text" },
        progress: { title: "进度", type: "number", column: { width: 130, component: { name: "a-progress", vModel: "percent", size: "small" } } },
        stage: { title: "阶段", type: "text", column: { width: 120 } },
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
        error: { title: "错误信息", type: "text", column: { width: 220, show: false } },
        startedAt: { title: "开始时间", type: "datetime", column: { width: 155 } },
        finishedAt: { title: "结束时间", type: "datetime", column: { width: 155 } }
      } as any,
      rowHandle: {
        fixed: "right",
        width: 130,
        buttons: {
          view: { show: false },
          edit: { show: false },
          copy: { show: false },
          log: { text: "", title: "查看恢复日志", type: "link", icon: "ion:document-text-outline", click: ({ row }: { row: RestoreTask }) => showLog(row) }
        }
      }
    }
  };
}
