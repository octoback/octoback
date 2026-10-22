import { CreateCrudOptionsRet } from "@fast-crud/fast-crud";
import { getAuditLogPage, type AuditLog } from "./api";
export function createCrudOptions(): CreateCrudOptionsRet<AuditLog> {
  return {
    crudOptions: {
      request: {
        pageRequest: ({ query, page }: any) => {
          return getAuditLogPage({ offset: page.offset, limit: page.limit, username: query?.username, path: query?.path });
        }
      },
      rowHandle: { show: false },
      columns: {
        id: { title: "Id", type: "text", column: { width: 100 } },
        operation: { title: "操作", type: "text", column: { width: 200 } },
        username: { title: "用户", type: "text", search: { show: true }, column: { width: 100 } },
        path: { title: "路径", type: "text", search: { show: true }, column: { width: 250 } },
        ip: { title: "IP 地址", type: "text", column: { width: 100 } },
        duration: { title: "耗时(ms)", type: "number" },
        createdAt: { title: "时间", type: "datetime" }
      } as any
    }
  };
}
