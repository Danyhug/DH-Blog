<template>
  <router-view></router-view>
</template>
<script>
import { heartBeat } from './api/user';
import { useAdminStore, useUserStore } from './store';

const timer = setInterval(() => {
  if (useUserStore().isBan) {
    // 停止心跳
    clearInterval(timer)
  } else {
    heartBeat().then(r => {
      useAdminStore().online = r.split('~')[2]
    })
  }
}, 5000);
</script>
<style></style>