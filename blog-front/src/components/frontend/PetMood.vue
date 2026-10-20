<template>
  <!--
    桌宠的心里话：只读 usePetMood 里的状态，不碰素材包，也不引用 Pet.vue。
    素材更新替换 Pet.vue 时，这个文件不需要跟着改 —— 位置是 Pet.vue 写进 usePetMood 的锚点，
    这里只管照着锚点把自己摆到她头顶。
  -->
  <div
    v-show="bubble.visible"
    class="pointer-events-none absolute z-10 -translate-x-1/2"
    :style="{ left: `clamp(5rem, ${bubble.x}px, calc(100% - 5rem))`, bottom: `${bubble.bottom}px` }"
    role="status"
    aria-live="polite"
  >
    <!--
      气泡尾巴：把同一套 glass-thick 材质转 45° 贴在卡片下沿，只露出下面那个尖角。
      用同一套材质而不是自己配颜色，是为了日夜两套主题下都和卡片一模一样；
      只留底边、右边两条描边（旋转后正好是 V 字的两笔），免得卡片上多出一圈方块线。
      描边颜色跟卡片一致；`!` 是因为 glass-thick 里写了 border 简写，普通 width 工具类压不过它。
    -->
    <span
      class="glass-thick absolute -bottom-2 left-1/2 h-4 w-4 -translate-x-1/2 rotate-45 rounded-[3px] border-t-0! border-l-0! border-grey-5/75 dark:border-white/30"
    ></span>
    <!-- 描边：glass-thick 默认是 70% 白，压在白色侧栏上等于没有；灰 5 的 75% 约等于 #b3b3b3，刚好看得清又不抢戏 -->
    <p class="glass-thick relative flex w-max max-w-[11rem] min-w-[4.5rem] items-start gap-1.5 rounded-2xl border-grey-5/75 px-3 py-2 dark:border-white/30">
      <!-- 唱歌时气泡换成歌词：上一行是唱到的那句，下一行小字是中文 -->
      <template v-if="singing">
        <span class="animate-pulse text-[15px] leading-[1.4]">♪</span>
        <span class="flex min-w-0 flex-col gap-0.5">
          <span class="text-[12px] leading-[1.35] text-grey-7 dark:text-[#d4d4d4]">{{ lyric }}</span>
          <span class="text-[10px] leading-[1.3] text-grey-5 dark:text-grey-4">{{ translation }}</span>
        </span>
      </template>
      <template v-else>
        <span class="text-[15px] leading-[1.4]">{{ mood.emoji }}</span>
        <span class="text-[12px] leading-[1.4] text-grey-7 dark:text-[#d4d4d4]">{{ mood.text }}</span>
      </template>
    </p>
  </div>
</template>

<script setup lang="ts">
import { usePetMood } from "@/composables/usePetMood";
import { usePetSing } from "@/composables/usePetSing";

const { mood, bubble } = usePetMood();
const { singing, lyric, translation } = usePetSing();
</script>
