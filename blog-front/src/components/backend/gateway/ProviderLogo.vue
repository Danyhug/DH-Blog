<template>
    <!-- 认识的供应商用打包进来的官方 logo；不认识的（新加了后端还没配图）降级为首字母色块 -->
    <img v-if="logo" :src="logo" :alt="name" class="rounded object-contain shrink-0" :style="box" />
    <span v-else class="rounded shrink-0 inline-flex items-center justify-center font-semibold text-white"
        :style="[box, { backgroundColor: fallbackColor(name), fontSize: `${Math.round(size * 0.5)}px` }]">
        {{ name.charAt(0).toUpperCase() }}
    </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { fallbackColor } from './format';
import braveLogo from '@/assets/images/providers/brave.svg';
import exaLogo from '@/assets/images/providers/exa.svg';
import firecrawlLogo from '@/assets/images/providers/firecrawl.svg';
import tavilyLogo from '@/assets/images/providers/tavily.svg';

/**
 * logo 是静态资源而不是后端下发的外链。
 * 外链意味着每次打开后台都要向四家第三方各拉一张图，对方换了地址就满屏破图；
 * 打包进来则跟着构建产物走，离线也能显示。新增供应商时在这里补一条即可。
 */
const LOGOS: Record<string, string> = {
    brave: braveLogo,
    tavily: tavilyLogo,
    exa: exaLogo,
    firecrawl: firecrawlLogo
};

const props = withDefaults(defineProps<{ name: string; size?: number }>(), {
    size: 24
});

const logo = computed(() => LOGOS[props.name?.toLowerCase()]);
const box = computed(() => ({ width: `${props.size}px`, height: `${props.size}px` }));
</script>
