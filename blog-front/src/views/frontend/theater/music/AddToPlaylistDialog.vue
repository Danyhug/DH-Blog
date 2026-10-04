<template>
  <el-dialog v-model="visible" title="添加到歌单" width="420px" append-to-body align-center class="glass-popup" modal-class="backdrop-blur-md">
    <div class="flex flex-col gap-1">
      <button :class="rowClass" @click="createAndAdd">
        <span class="flex size-11 items-center justify-center rounded-md bg-black/5 dark:bg-white/10 text-[#fa2d48]"><PlusIcon class="size-5" /></span>
        <span class="font-medium text-zinc-900 dark:text-white">新建歌单</span>
      </button>
      <button v-for="playlist in music.playlists" :key="playlist.id" :class="rowClass" :disabled="busy" @click="addTo(playlist)">
        <Artwork :src="coverOf(playlist)" :seed="playlist.name" class="size-11 shrink-0 rounded-md" />
        <span class="min-w-0 flex-1">
          <span class="block truncate font-medium text-zinc-900 dark:text-white">{{ playlist.name }}</span>
          <span class="block text-xs text-black/50 dark:text-white/50">{{ playlist.track_ids.length }} 首歌曲</span>
        </span>
        <CheckIcon v-if="containsAll(playlist)" class="size-4 text-[#fa2d48]" />
      </button>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useMusicStore } from '@/store'
import { trackCoverUrl, type Playlist, type Track } from '@/api/media'
import { notify } from '@/utils/notification'
import Artwork from '../components/Artwork.vue'
import { CheckIcon, PlusIcon } from '../components/icons'

const props = defineProps<{ tracks: Track[] }>()
const visible = defineModel<boolean>({ required: true })
const music = useMusicStore()
const busy = ref(false)

const rowClass = 'flex w-full cursor-pointer items-center gap-3 rounded-xl border-none bg-transparent p-2 text-left text-sm transition-colors hover:bg-black/5 dark:hover:bg-white/10 disabled:cursor-wait'

function coverOf(playlist: Playlist) {
  const first = music.playlistTracks(playlist).find(track => track.cover_file_id || track.has_embedded_cover)
  return first ? trackCoverUrl(first) : ''
}

function containsAll(playlist: Playlist) {
  return props.tracks.every(track => playlist.track_ids.includes(track.id))
}

async function addTo(playlist: Playlist) {
  busy.value = true
  try {
    await music.addToPlaylist(playlist, props.tracks.map(track => track.id))
    notify.success(`已添加到「${playlist.name}」`)
    visible.value = false
  } finally {
    busy.value = false
  }
}

async function createAndAdd() {
  const { value } = await ElMessageBox.prompt('歌单名称', '新建歌单', {
    confirmButtonText: '创建',
    cancelButtonText: '取消',
    inputValue: '新建歌单',
    customClass: 'glass-popup',
    inputValidator: v => (v.trim() ? true : '名称不能为空')
  }).catch(() => ({ value: '' }))
  if (!value.trim()) return
  const playlist = await music.addPlaylist(value.trim(), props.tracks.map(track => track.id))
  notify.success(`已创建歌单「${playlist.name}」`)
  visible.value = false
}
</script>
