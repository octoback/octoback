<template>
  <PluginCrud plugin-type="repository" title="存储仓库" description="配置 Restic 使用的 SFTP 和阿里云 OSS 仓库。" />
</template>

<script setup lang="ts">
defineOptions({
  name: "BackupRepository"
});

import PluginCrud from "/@/views/sys/plugin/index.vue";
</script>
