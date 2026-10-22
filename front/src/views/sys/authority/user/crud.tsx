import { AddReq, CreateCrudOptionsProps, CreateCrudOptionsRet, DelReq, dict, EditReq } from "@fast-crud/fast-crud";
import { createUser, deleteUser, getUserList, updateUser } from "./api";

export default function (_props: CreateCrudOptionsProps): CreateCrudOptionsRet {
  return {
    crudOptions: {
      request: {
        pageRequest: getUserList,
        addRequest: async ({ form }: AddReq) => createUser({ ...form, role: form.role || "read" }),
        editRequest: async ({ form, row }: EditReq) => updateUser(row.id, form),
        delRequest: async ({ row }: DelReq) => deleteUser(row.id)
      },
      rowHandle: { fixed: "right" },
      columns: {
        id: { title: "Id", type: "text", form: { show: false }, column: { width: 80 } },
        username: { title: "用户名", type: "text", search: { show: true }, form: { rules: [{ required: true, message: "请输入用户名" }] }, editForm: { component: { disabled: true } } },
        password: { title: "密码", type: "text", column: { show: false }, form: { rules: [{ min: 6, message: "密码至少 6 位" }], component: { showPassword: true }, helper: "编辑时留空表示不修改" } },
        nickName: { title: "昵称", type: "text" },
        role: {
          title: "角色",
          type: "dict-select",
          dict: dict({
            data: [
              { value: "admin", label: "管理员" },
              { value: "write", label: "读写" },
              { value: "read", label: "只读" }
            ]
          }),
          form: { rules: [{ required: true, message: "请选择角色" }] }
        },
        createdAt: { title: "创建时间", type: "datetime", form: { show: false } }
      }
    }
  };
}
