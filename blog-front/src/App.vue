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
<script setup>
import { watchEffect } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const reading = useUserStore().aritcleModel
// html.dark switches both Tailwind's dark: variant and Element Plus's dark CSS vars.
// Routes opt in with meta.blogTheme (blog, drive, share, theater), so admin and login stay light whatever the reader picked.
// Pre-flush runs before the next render, which the theme toggle's view transition awaits via nextTick.
watchEffect(() => {
  document.documentElement.classList.toggle('dark', reading.isDarkMode && route.meta.blogTheme === true)
})
</script>
