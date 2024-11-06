<template>
  <router-view></router-view>
</template>
<script>
import { heartBeat } from './api/user';
import { useAdminStore } from './store';

setInterval(() => {
  heartBeat().then(r => {
    useAdminStore().online = r.split('~')[2]
  })
}, 5000);
</script>
<style></style>