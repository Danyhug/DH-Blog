import { reactive, readonly } from "vue";

/**
 * 桌宠「心情」：Pet.vue 负责写（她做了什么、现在什么配色），PetMood.vue 负责画。
 *
 * 单独抽一个模块而不是塞进 Pet.vue，是因为 Pet.vue 只做素材包的宿主：素材更新时整个文件会被替换，
 * 心情框的 UI 和文案不应该跟着重写。两个组件各自 import 这里，谁先挂载都不影响。
 *
 * 这里同时放了她头顶的锚点：气泡要贴着她走，但画在哪由 PetMood.vue 决定，
 * 素材包换掉之后只要新版本还在写锚点，气泡位置就还是对的。
 */
export interface PetMoodState {
  /** 心情表情，一个 emoji */
  emoji: string;
  /** 一句话，显示在气泡里 */
  text: string;
  /** 当前配色（素材包里的 scheme id 与显示名） */
  schemeId: string;
  schemeName: string;
  /** 本次加载抽到的服装 */
  outfitId: string;
  outfitName: string;
  /** 最后一次更新心情的时间戳，用于判断要不要淡出 */
  at: number;
}

const IDLE: PetMoodState = {
  emoji: "🫧",
  text: "发呆中…",
  schemeId: "",
  schemeName: "配色加载中",
  outfitId: "",
  outfitName: "",
  at: 0,
};

// 模块级单例：Pet.vue 与 PetMood.vue 通过它共享同一份状态
const shared = reactive<PetMoodState>({ ...IDLE });
const mood = readonly(shared);

/** 她头顶那一点，单位是舞台像素（舞台 = Pet.vue 的可视区域） */
export interface PetBubbleAnchor {
  /** 横坐标，气泡以它居中 */
  x: number;
  /** 舞台底边到气泡底边的距离，用 bottom 定位免得气泡高度一变就往下压到她头上 */
  bottom: number;
  /** 素材还没挂上、或桌宠已经卸载时不显示 */
  visible: boolean;
}

const sharedBubble = reactive<PetBubbleAnchor>({ x: 0, bottom: 0, visible: false });
const bubble = readonly(sharedBubble);

export const usePetMood = () => {
  const setPetMood = (patch: Partial<Omit<PetMoodState, "at">>) => {
    Object.assign(shared, patch, { at: Date.now() });
  };

  const setPetBubble = (patch: Partial<PetBubbleAnchor>) => {
    Object.assign(sharedBubble, patch);
  };

  const resetPetMood = () => setPetMood(IDLE);

  return { mood, bubble, setPetMood, setPetBubble, resetPetMood };
};
