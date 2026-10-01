<template>
  <div role="toolbar" aria-label="文章阅读工具" class="flex items-center justify-evenly gap-1"
    :class="model.isDarkMode ? 'text-[#b8b8b8]' : 'text-[#606266]'">
    <button type="button" :class="buttonClass" :title="focusLabel" :aria-label="focusLabel"
      :aria-pressed="model.isFullPreview" @click="model.isFullPreview = !model.isFullPreview">
      <Close v-if="model.isFullPreview" class="size-5" />
      <FullScreen v-else class="size-5" />
    </button>
    <el-popover v-model:visible="qrPanelOpen" trigger="click" placement="bottom" :width="240"
      :persistent="false" :effect="model.isDarkMode ? 'dark' : 'light'">
      <template #reference>
        <button type="button" :class="buttonClass" title="生成二维码" aria-label="生成文章二维码">
          <Icon iconName="icon-31erweima" iconSize="1.25" aria-hidden="true" />
        </button>
      </template>
      <div class="flex flex-col items-center gap-2" role="group" aria-label="文章二维码">
        <QrcodeVue :value="articleUrl" :size="200" :margin="4" level="M" render-as="svg"
          background="#ffffff" foreground="#000000" role="img" aria-label="当前文章链接二维码" />
        <p class="m-0 text-center text-[13px]">扫码阅读这篇文章</p>
      </div>
    </el-popover>
    <el-popover v-model:visible="fontPanelOpen" trigger="click" placement="bottom" :width="220"
      :persistent="false" :effect="model.isDarkMode ? 'dark' : 'light'">
      <template #reference>
        <button type="button" :class="buttonClass" title="调整字号" aria-label="调整字号">
          <span class="text-[18px] font-medium" aria-hidden="true">Aa</span>
        </button>
      </template>
      <div class="flex flex-col gap-3" role="group" aria-label="正文字号">
        <div class="flex items-center justify-between">
          <span>正文字号</span>
          <output aria-live="polite">{{ model.fontSize }}px</output>
        </div>
        <div class="flex items-center justify-between gap-2">
          <button type="button" :class="buttonClass" aria-label="减小字号" :disabled="model.fontSize <= 14"
            @click="model.fontSize = Math.max(14, model.fontSize - 2)"><Minus class="size-4" /></button>
          <button type="button" class="cursor-pointer rounded px-2 py-1 text-[13px] hover:bg-black/5 focus-visible:outline-2 focus-visible:outline-blue-500"
            @click="model.fontSize = 16">恢复默认</button>
          <button type="button" :class="buttonClass" aria-label="增大字号" :disabled="model.fontSize >= 24"
            @click="model.fontSize = Math.min(24, model.fontSize + 2)"><Plus class="size-4" /></button>
        </div>
      </div>
    </el-popover>
    <button type="button" :class="buttonClass" :title="themeLabel" :aria-label="themeLabel"
      :aria-pressed="model.isDarkMode" @click="toggleDarkMode">
      <Sunny v-if="model.isDarkMode" class="size-5" />
      <Moon v-else class="size-5" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { Close, FullScreen, Minus, Moon, Plus, Sunny } from '@element-plus/icons-vue';
import { useRoute } from 'vue-router';
import { useUserStore } from '@/store';
import QrcodeVue from 'qrcode.vue';

const model = useUserStore().aritcleModel;
const route = useRoute();
const fontPanelOpen = ref(false);
const qrPanelOpen = ref(false);
const articleUrl = computed(() => {
  const url = new URL(window.location.href);
  // Encode only the article route, never login tokens or other query state.
  url.search = '';
  url.hash = `/view/article/${encodeURIComponent(String(route.params.id))}`;
  return url.href;
});
watch(qrPanelOpen, (visible) => { if (visible) fontPanelOpen.value = false; });
watch(fontPanelOpen, (visible) => { if (visible) qrPanelOpen.value = false; });
// Both toolbar instances can teleport a popover; close it before switching layouts.
watch([() => model.isFullPreview, () => route.fullPath], () => {
  fontPanelOpen.value = false;
  qrPanelOpen.value = false;
});
const focusLabel = computed(() => model.isFullPreview ? '退出专注阅读' : '专注阅读');
const themeLabel = computed(() => model.isDarkMode ? '切换日间模式' : '切换夜间模式');
// Reveal the new theme as a circle growing from the button (styles in tailwind.css).
const toggleDarkMode = (event: MouseEvent) => {
  const flip = () => { model.isDarkMode = !model.isDarkMode; };
  // element.animate() ignores the global reduced-motion CSS, so honour the preference here.
  if (typeof document.startViewTransition !== 'function' || matchMedia('(prefers-reduced-motion: reduce)').matches) {
    flip();
    return;
  }
  // Use the button's centre rather than the pointer: keyboard activation reports 0,0.
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;
  const radius = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y));
  const transition = document.startViewTransition(async () => {
    flip();
    await nextTick();
  });
  transition.ready.then(() => {
    document.documentElement.animate(
      { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${radius}px at ${x}px ${y}px)`] },
      { duration: 450, easing: 'ease-in-out', pseudoElement: '::view-transition-new(root)' },
    );
  }).catch(() => {
    // Transition was skipped (e.g. tab hidden); the theme has still switched.
  });
};
const buttonClass = computed(() => [
  'inline-flex size-11 shrink-0 items-center justify-center rounded-lg border-0 bg-transparent cursor-pointer focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 disabled:opacity-40 disabled:cursor-not-allowed',
  model.isDarkMode ? 'hover:bg-white/10' : 'hover:bg-black/5',
]);

</script>
