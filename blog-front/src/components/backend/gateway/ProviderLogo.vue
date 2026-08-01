<template>
    <!-- logo 直接引用各家官网资源，加载失败时降级为首字母色块，避免外链挂掉后满屏破图 -->
    <img v-if="src && !broken" :src="src" :alt="name" class="rounded object-contain shrink-0" :style="box" loading="lazy"
        @error="broken = true" />
    <span v-else class="rounded shrink-0 inline-flex items-center justify-center font-semibold text-white"
        :style="[box, { backgroundColor: fallbackColor(name), fontSize: `${Math.round(size * 0.5)}px` }]">
        {{ name.charAt(0).toUpperCase() }}
    </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { fallbackColor } from './format';

const props = withDefaults(defineProps<{ name: string; src?: string; size?: number }>(), {
    src: '',
    size: 24
});

const broken = ref(false);
// 表格会复用同一个组件实例渲染不同的供应商，换了地址就得重新给一次加载机会
watch(() => props.src, () => { broken.value = false; });

const box = computed(() => ({ width: `${props.size}px`, height: `${props.size}px` }));
</script>
