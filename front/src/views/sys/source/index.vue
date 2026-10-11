<template>
  <PluginCrud plugin-type="source" title="数据资源" description="配置 SSH 主机上需要备份的目录。" />
</template>

<script setup lang="ts">
defineOptions({
  name: "BackupSource"
});

import PluginCrud from "/src/views/sys/plugin/index.vue";
</script>
