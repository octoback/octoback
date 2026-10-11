import { AddReq, CreateCrudOptionsRet, DelReq, EditReq, dict } from "@fast-crud/fast-crud";
import { createBackupPlan, deleteBackupPlan, getBackupPlanInfo, getBackupPlanPage, updateBackupPlan, type BackupPlan } from "./api";
function policyFromPlan(plan: BackupPlan) {
  return { ...plan, retentionPolicy: plan.keepPolicy || { type: "none", value: {} } };
}
function planFromForm(form: any) {
  const result = { ...form };
  result.keepPolicy = result.retentionPolicy || { type: "none", value: {} };
  delete result.retentionPolicy;
  return result;
}
export default function (): CreateCrudOptionsRet<BackupPlan> {
  return {
    crudOptions: {
      request: {
        pageRequest: ({ query }: any) => getBackupPlanPage({ offset: query.offset, limit: query.limit, name: query.query?.name }),
        infoRequest: ({ row, mode }: any) => {
          if (mode === "add") {
            return {};
          }
          return getBackupPlanInfo(row.id).then(policyFromPlan);
        },
        addRequest: ({ form }: AddReq<BackupPlan>) => createBackupPlan(planFromForm(form)),
        editRequest: ({ form, row }: EditReq<BackupPlan>) => updateBackupPlan(row.id, planFromForm(form)),
        delRequest: ({ row }: DelReq<BackupPlan>) => deleteBackupPlan(row.id)
      },
      form: {
        wrapper: {
          width: "1000px" //antdv对话框的宽度
        }
      },
      columns: {
        id: { title: "ID", type: "text", form: { show: false } },
        name: { title: "计划名称", type: "text", search: { show: true }, form: { helper: "备份计划名称", rules: [{ required: true, message: "请输入名称" }] } },
        sourceId: { title: "数据资源", type: "text", form: { component: { name: "source-selector", vModel: "modelValue" }, helper: "备份数据来源", rules: [{ required: true, message: "请选择数据资源" }] } },
        repositoryId: {
          title: "存储仓库",
          type: "text",
          form: { component: { name: "repository-selector", vModel: "modelValue" }, helper: "备份数据存储到哪个仓库", rules: [{ required: true, message: "请选择存储仓库" }] }
        },
        repoTag: { title: "仓库标签", type: "text", form: { component: { placeholder: "可选，用于区分同一仓库中的备份" }, helper: "使用 Restic 标签区分备份计划" } },
        retentionPolicy: {
          title: "保留策略",
          type: "text",
          form: {
            component: { name: "retention-policy", vModel: "modelValue" },
            helper: "按 Restic 快照数量和周期清理，全部为 0 表示不启用",
            rules: [
              {
                validator: (_rule: unknown, value: any) => {
                  if (!value || value.type === "none") {
                    return Promise.resolve();
                  }
                  if (value.type === "time") {
                    const duration = value.value?.duration;
                    if (!duration) {
                      return Promise.reject(new Error("按时间策略不能为空"));
                    }
                    if (!/^\d+$/.test(String(duration).trim())) {
                      return Promise.reject(new Error("请输入有效的时间，例如 30d 或 12h"));
                    }
                    return Promise.resolve();
                  }
                  if (!Object.values(value.value || {}).some((item) => Number(item || 0) > 0)) {
                    return Promise.reject(new Error("按数量策略至少填写一个大于 0 的值"));
                  }
                  return Promise.resolve();
                }
              }
            ]
          }
        },
        compression: {
          title: "备份压缩",
          type: "dict-select",
          form: { helper: "Restic 压缩级别" },
          dict: dict({
            data: [
              { value: "auto", label: "自动" },
              { value: "off", label: "关闭" },
              { value: "max", label: "最大" }
            ]
          })
        },
        schedule: {
          title: "Cron 调度",
          type: "text",
          form: {
            component: { name: "cron-editor", vModel: "modelValue", allowEveryMin: true },
            rules: [
              { required: true, message: "请选择 Cron 调度" },
              { pattern: /^\S+(\s+\S+){4}$/, message: "请输入 5 段式 Cron 表达式" }
            ]
          }
        },
        enabled: {
          title: "计划启用",
          type: "dict-switch",
          column: { show: false },
          dict: dict({
            data: [
              { value: true, label: "启用" },
              { value: false, label: "禁用" }
            ]
          })
        },
        lastStatus: {
          title: "运行状态",
          type: "dict-select",
          dict: dict({
            data: [
              { value: "pending", label: "待执行" },
              { value: "running", label: "执行中" },
              { value: "success", label: "成功" },
              { value: "failed", label: "失败" }
            ]
          }),
          form: { show: false }
        },
        createdAt: { title: "创建时间", type: "datetime", form: { show: false } }
      } as any,
      rowHandle: { fixed: "right" }
    }
  };
}
