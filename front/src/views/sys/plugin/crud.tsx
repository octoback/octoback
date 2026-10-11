import { AddReq, compute, CreateCrudOptionsProps, CreateCrudOptionsRet, DelReq, dict, EditReq } from "@fast-crud/fast-crud";
import { usePluginDefineStore } from "/src/store/modules/plugin-define";
import { createPluginInstance, deletePluginInstance, getPluginInstanceInfo, getPluginInstancePage, type PluginField, type PluginInstance, type PluginMetadata, updatePluginInstance } from "./plugin-api";

export type PluginCrudContext = { pluginType: string; pluginName?: string; metadata: PluginMetadata[] };

export function setPluginAccessType(form: Record<string, any>, accessType?: string, resetAccess = false) {
  form.config ||= {};
  if (resetAccess && form.config.accessType && form.config.accessType !== accessType) {
    form.config.accessId = undefined;
  }
  form.config.accessType = accessType;
}

function getComponent(field: PluginField): Record<string, unknown> {
  const configured = { ...(field.component || {}) } as Record<string, unknown>;
  if (field.mergeScript) {
    const script = new Function("ctx", field.mergeScript) as (ctx: { compute: typeof compute }) => { component?: Record<string, unknown> };
    Object.assign(configured, script({ compute }).component || {});
  }
  const name = (configured.name || "a-input") as string;
  const vModel = name.startsWith("a-") ? "value" : "modelValue";
  return {
    vModel,
    ...configured,
    name
  };
}

export default function ({ context, crudExpose }: CreateCrudOptionsProps<PluginInstance, PluginCrudContext>): CreateCrudOptionsRet<PluginInstance> {
  const { metadata, pluginType } = context;
  let pluginTypeTitle = "插件类型";
  if (pluginType === "access") {
    pluginTypeTitle = "授权类型";
  }
  if (pluginType === "repository") {
    pluginTypeTitle = "存储类型";
  }
  if (pluginType === "source") {
    pluginTypeTitle = "资源类型";
  }
  const pluginDefineStore = usePluginDefineStore();
  const dynamicFieldKeys = new Set<string>();
  const goRepository = (row: PluginInstance) => (window.location.hash = `#/sys/repository/${row.id}`);
  const columns: Record<string, any> = {
    id: { title: "ID", type: "text", form: { show: false }, column: { width: 80 } },
    name: {
      title: "名称",
      type: "text",
      order: -11,
      search: { show: true },
      column: {
        sorter: false,
        conditionalRender: false,
        cellRender: ({ row }: { row: PluginInstance }) => {
          if (pluginType === "repository") {
            return (
              <a class="text-primary" onClick={() => goRepository(row)}>
                {row.name}
              </a>
            );
          }
          return <span>{row.name}</span>;
        }
      },
      form: { order: -11, rules: [{ required: true, message: "请输入名称" }] }
    },
    pluginName: {
      title: pluginTypeTitle,
      type: "dict-select",
      order: -10,
      dict: dict({
        getData: async () => {
          const defines = await pluginDefineStore.init();
          return defines.filter((item) => item.type === pluginType).map((item) => ({ value: item.name, label: item.title }));
        }
      }),
      addForm: { component: { disabled: context.pluginName != null } },
      editForm: { component: { disabled: true } },
      form: {
        order: -10,
        rules: [{ required: true, message: `请选择${pluginTypeTitle}` }],
        component: {
          showSearch: true,
          on: {
            selectedChange: ({ form }: { form: Record<string, any> }) => {
              setPluginAccessType(form, metadata.find((item) => item.name === form.pluginName)?.accessType, true);
            }
          }
        },
        valueChange: {
          immediate: true,
          handle: ({ value, form, mode, immediate }: { value: string; form: Record<string, any>; mode: string; immediate: boolean }) => {
            buildDynamicFields(value, form, mode);
            setPluginAccessType(form, metadata.find((item) => item.name === value)?.accessType, !immediate);
          }
        }
      }
    },
    createdAt: { title: "创建时间", type: "datetime", form: { show: false } }
  };

  function buildDynamicFields(pluginName: string, form: Record<string, any>, mode: string) {
    const formWrapper = crudExpose.getFormWrapperRef();
    const formColumns = formWrapper?.formOptions?.columns;
    if (!formColumns) {
      return;
    }
    console.log('crudBinding.value[mode + "Form"].columns', formColumns);
    const define = metadata.find((item) => item.name === pluginName);
    if (!define) {
      return;
    }
    (define.fields || []).forEach((field, index) => {
      const component = getComponent(field);
      const isActionComponent = component.name === "plugin-test-button";
      const key = isActionComponent ? "__pluginField_" + field.key : "config." + field.key;
      const fieldColumn: Record<string, any> = {
        title: field.title,
        type: field.type || "text",
        order: 10,
        ...field,
        key,
        component,
        form: field.helper ? { helper: field.helper } : undefined
      };
      if (isActionComponent) {
        fieldColumn.form = { show: true };
      }
      if (field.options?.length && component.name === "a-select") {
        fieldColumn.form.component = { ...component, options: field.options };
      }
      formColumns[key] = fieldColumn;
      dynamicFieldKeys.add(key);
    });
    console.log('crudBinding.value[mode + "Form"].columns', formColumns);
  }

  return {
    crudOptions: {
      request: {
        infoRequest: async ({ row, mode }: { row: PluginInstance; mode: string }) => {
          if (mode === "add") {
            return { pluginName: context.pluginName };
          }
          return await getPluginInstanceInfo(row.id);
        },
        pageRequest: async (query: any) => {
          const limit = query.limit ?? 20;
          const offset = query.offset;
          return getPluginInstancePage({ offset, limit, pluginType, pluginName: context.pluginName, name: query.query?.name });
        },
        addRequest: ({ form }: AddReq) => createPluginInstance({ name: form.name, description: form.description || "", pluginType, pluginName: form.pluginName, config: form.config || {} }),
        editRequest: ({ form, row }: EditReq) => updatePluginInstance(row.id, { name: form.name, description: form.description || "", config: form.config || {} }),
        delRequest: async ({ row }: DelReq) => deletePluginInstance(row.id)
      },
      rowHandle: { fixed: "right" },
      columns: columns as any
    }
  };
}
