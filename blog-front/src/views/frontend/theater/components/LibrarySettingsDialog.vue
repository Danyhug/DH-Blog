<template>
  <el-dialog v-model="visible" title="媒体库设置" width="560px" append-to-body align-center class="glass-popup" modal-class="backdrop-blur-md" @open="load">
    <div v-loading="loading" class="flex flex-col gap-6">
      <p class="m-0 text-sm leading-relaxed text-zinc-500 dark:text-[#a3a3a3]">
        默认自动识别整个网盘里的音频和视频。如果网盘里杂物较多，可以只指定几个文件夹作为音乐库或影视库（包含子文件夹）。
      </p>

      <section v-for="section in sections" :key="section.kind" class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <h4 class="m-0 flex items-center gap-2 text-[15px] font-semibold text-zinc-900 dark:text-white">
            <component :is="section.icon" class="size-4" :class="section.accent" />
            {{ section.title }}
          </h4>
          <el-button size="small" :type="picking === section.kind ? 'primary' : 'default'" @click="togglePicker(section.kind)">
            {{ picking === section.kind ? '完成选择' : '添加文件夹' }}
          </el-button>
        </div>
        <div class="flex flex-wrap gap-2">
          <span v-if="!selected[section.kind].length" class="text-sm text-zinc-400 dark:text-[#737373]">未指定，自动扫描整个网盘</span>
          <el-tag v-for="folder in selected[section.kind]" :key="folder.id" closable @close="remove(section.kind, folder.id)">
            {{ folder.path || folder.name }}
          </el-tag>
        </div>
        <div v-if="picking === section.kind" class="glass max-h-64 overflow-auto rounded-2xl p-2 [--el-tree-bg-color:transparent] [--el-fill-color-blank:transparent]">
          <el-tree lazy :load="loadNode" :props="treeProps" node-key="id" :expand-on-click-node="false" @node-click="(data: TreeFolder, node: TreeNode) => add(section.kind, data, node)">
            <template #default="{ data }">
              <span class="flex items-center gap-2 text-sm">
                <FolderIcon class="size-4 text-[#e5a50a]" />
                {{ data.name }}
                <CheckIcon v-if="isSelected(section.kind, data.id)" class="size-4 text-[#46d369]" />
              </span>
            </template>
          </el-tree>
        </div>
      </section>
    </div>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { listFiles, type FileInfo } from '@/api/file'
import { getLibrarySettings, saveLibrarySettings, type FolderRef } from '@/api/media'
import { notify } from '@/utils/notification'
import { CheckIcon, FilmIcon, FolderIcon, MusicNoteIcon } from './icons'

type Kind = 'music' | 'video'
interface TreeFolder { id: string; name: string; leaf?: boolean }
interface TreeNode { level: number; data: TreeFolder; parent: TreeNode | null }

const visible = defineModel<boolean>({ required: true })
const emit = defineEmits<{ (e: 'saved'): void }>()

const sections = [
  { kind: 'music' as const, title: '音乐库', icon: MusicNoteIcon, accent: 'text-[#fa2d48]' },
  { kind: 'video' as const, title: '影视库', icon: FilmIcon, accent: 'text-[#e50914]' }
]
const treeProps = { label: 'name', isLeaf: 'leaf' }

const loading = ref(false)
const saving = ref(false)
const picking = ref<Kind | null>(null)
const selected = reactive<Record<Kind, FolderRef[]>>({ music: [], video: [] })

async function load() {
  loading.value = true
  picking.value = null
  try {
    const settings = await getLibrarySettings()
    selected.music = settings.music_folders
    selected.video = settings.video_folders
  } finally {
    loading.value = false
  }
}

async function loadNode(node: TreeNode, resolve: (data: TreeFolder[]) => void) {
  try {
    const parentId = node.level === 0 ? '' : node.data.id
    const files: FileInfo[] = (await listFiles(parentId)) || []
    resolve(files.filter(file => file.is_folder).map(file => ({ id: String(file.id), name: file.name })))
  } catch {
    resolve([])
  }
}

function pathOf(node: TreeNode): string {
  const names: string[] = []
  for (let current: TreeNode | null = node; current && current.level > 0; current = current.parent) names.unshift(current.data.name)
  return names.join('/')
}

function togglePicker(kind: Kind) {
  picking.value = picking.value === kind ? null : kind
}

function isSelected(kind: Kind, id: string) {
  return selected[kind].some(folder => folder.id === id)
}

function add(kind: Kind, data: TreeFolder, node: TreeNode) {
  if (isSelected(kind, data.id)) return remove(kind, data.id)
  selected[kind].push({ id: data.id, name: data.name, path: pathOf(node) })
}

function remove(kind: Kind, id: string) {
  selected[kind] = selected[kind].filter(folder => folder.id !== id)
}

async function save() {
  saving.value = true
  try {
    await saveLibrarySettings(selected.music.map(folder => folder.id), selected.video.map(folder => folder.id))
    notify.success('媒体库设置已保存')
    visible.value = false
    emit('saved')
  } finally {
    saving.value = false
  }
}
</script>
