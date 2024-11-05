<template>
  <router-view></router-view>
</template>
<script>
import { heartBeat } from './api/user';

setInterval(() => heartBeat(), 5000);
</script>
<style>
</style>