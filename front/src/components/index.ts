import { defineAsyncComponent } from "vue";
import "@vue-js-cron/light/dist/light.css";

const AsyncHighLight = defineAsyncComponent(() => import("./highlight/index.vue"));
const AccessSelector = defineAsyncComponent(() => import("./access-selector/index.vue"));
const SourceSelector = defineAsyncComponent(() => import("./source-selector/index.vue"));
const RepositorySelector = defineAsyncComponent(() => import("./repository-selector/index.vue"));
const PluginSelector = defineAsyncComponent(() => import("./plugin-selector/index.vue"));
const PluginFileInput = defineAsyncComponent(() => import("./plugin-file-input/index.vue"));
const PluginPathSelector = defineAsyncComponent(() => import("./plugin-path-selector/index.vue"));
const CronEditor = defineAsyncComponent(() => import("./cron-editor/index.vue"));
const RetentionPolicy = defineAsyncComponent(() => import("./retention-policy/index.vue"));
const PluginTestButton = defineAsyncComponent(() => import("./plugin-test-button/index.vue"));
export default {
  install(app: any) {
    app.component("FsHighlight", AsyncHighLight);
    app.component("AccessSelector", AccessSelector);
    app.component("SourceSelector", SourceSelector);
    app.component("RepositorySelector", RepositorySelector);
    app.component("PluginSelector", PluginSelector);
    app.component("PluginFileInput", PluginFileInput);
    app.component("PluginPathSelector", PluginPathSelector);
    app.component("CronEditor", CronEditor);
    app.component("RetentionPolicy", RetentionPolicy);
    app.component("PluginTestButton", PluginTestButton);
  }
};
