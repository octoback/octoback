import { CreateCrudOptionsRet, dict } from "@fast-crud/fast-crud";
import { Modal } from "ant-design-vue";
import { h } from "vue";
import { getBackupPlanList } from "../api";
import { getBackupRunLog, getBackupRunPage, type BackupLog } from "../api";
export default function (): CreateCrudOptionsRet<BackupLog> {
  const openRepository = (row: BackupLog) => {
    if (row.repositoryId) {
      window.location.hash = `#/sys/repository/${row.repositoryId}`;
    }
  };
  const openResticLog = async (row: BackupLog) => {
    const response = (await getBackupRunLog(row.id)) as { content?: string };
    const content = response.content || "暂无 Restic 日志";
    Modal.info({
      title: `Restic 日志 - ${row.planName || `运行记录 ${row.id}`}`,
      width: 860,
      content: h(
        "pre",
        {
          style: {
            maxHeight: "560px",
            margin: 0,
            padding: "12px",
            overflow: "auto",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
            background: "#f6f8fa",
            borderRadius: "6px",
            fontFamily: "ui-monospace, SFMono-Regular, Consolas, monospace",
            fontSize: "12px",
            lineHeight: "1.55"
          }
        },
        content
      )
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
