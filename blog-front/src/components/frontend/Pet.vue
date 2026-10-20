<template>
  <!--
    素材包的宿主：只负责把 public/pet 里的 figure 挂上来、喂帧、转达交互。
    她做什么（随机动作）、说什么（心情）都通过 usePetMood 交给 PetMood.vue 去画，
    所以素材更新时这个文件可以整个替换掉。
  -->
  <div
    class="relative h-[13rem] w-full touch-none select-none xl:h-[16.5rem]"
    :style="stageStyle"
    @click="onClick"
    @dblclick="onRandomScheme"
    @pointerenter="onPointerEnter"
    @pointermove="onPointerMove"
  >
    <div ref="rootElement" class="absolute inset-0"></div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { usePetMood } from "@/composables/usePetMood";
import { usePetSing } from "@/composables/usePetSing";

/* ---------- 素材包（public/pet，运行期加载；更新素材不需要动这里） ---------- */

/** 素材包里的配色，双击时从中随机抽一套；schemeId 必须和 model.json 的 schemes 一致 */
const SCHEMES = [
  { id: "deepseek", name: "DeepSeek 蓝" },
  { id: "harness", name: "纯黑" },
  { id: "chatgpt", name: "银白" },
  { id: "claude", name: "赤陶" },
  { id: "gemini", name: "四色" },
  { id: "qwen", name: "千问紫" },
  { id: "kimi", name: "黑蓝" },
  { id: "minimax", name: "玫红" },
] as const;

/** 服装只有两套：女仆装是默认部件表，冬装是另一套整套部件 */
const OUTFITS = [
  { id: "maid", name: "女仆装" },
  { id: "cape", name: "冬装" },
] as const;

type PetOutfit = (typeof OUTFITS)[number];

/* ---------- kit / figure 的最小类型（包是运行期加载的普通 ESM，没有 d.ts） ---------- */

interface PetBounds {
  W: number;
  H: number;
  floorY: number;
  S: number;
}

interface PetStart {
  x: number;
  facing: 1 | -1;
  enter?: "drop";
}

interface PetHost {
  root: HTMLElement;
  bounds: () => PetBounds;
  emit: (kind: string, detail?: unknown) => void;
  sound: (...args: unknown[]) => void;
  /** kit 只在这里读一次：她从哪里落下、朝哪边站 */
  start: PetStart;
}

interface PetLayout {
  /** 舞台像素坐标里她脚下当前的位置 */
  x: number;
  /** 她身体的包围盒（舞台像素），用来区分点到她还是点到空地 */
  box: { x: number; y: number; w: number; h: number };
  /** 她头顶那一点（舞台像素），说话的气泡挂在这儿；跟着蹲下、跳起、转身一起动 */
  bubble: { x: number; y: number };
  mode: string;
  busy: boolean;
  moving: boolean;
}

interface PetBody {
  step: (dt: number) => void;
  layout: () => PetLayout;
  walk: (x: number, run: boolean, id: number) => boolean;
  stopWalk: (id: number) => void;
  do: (word: string) => void;
  set: (state: { roam?: "free" | "calm" | "off" }) => void;
  setScheme: (id: string, options?: { fade?: number }) => Promise<void> | void;
  resize: () => void;
  dispose: () => void;
}

interface PetKitModule {
  createBody: (host: PetHost, options: { figure: unknown }) => PetBody;
}

interface WhaleFigureModule {
  createWhaleBody: (
    base: URL,
    options: { kit: PetKitModule; host: PetHost; scheme: string }
  ) => Promise<PetBody>;
}

/* ---------- 她自己的小情绪：每个动作配一句 + 一个表情 ---------- */

interface PetAction {
  word: string;
  emoji: string;
  text: string;
}

/** 闲着时随机做的事，只挑不会让她长时间赖在某个姿态里的词（sit / sleep 这类不选） */
const ACTIONS: PetAction[] = [
  { word: "wave", emoji: "👋", text: "嗨，来看看文章～" },
  { word: "hop", emoji: "✨", text: "蹦一下！" },
  { word: "jump", emoji: "🐳", text: "跳得高不高？" },
  { word: "nod", emoji: "🙂", text: "嗯嗯，说得对" },
  { word: "shake", emoji: "🙅", text: "这个我可不同意" },
  { word: "spin", emoji: "💫", text: "原地转个圈" },
  { word: "dance", emoji: "🎵", text: "哼哼哼～" },
  { word: "bow", emoji: "🙇", text: "谢谢阅读" },
  { word: "flap", emoji: "🐚", text: "扑腾扑腾" },
  { word: "happy", emoji: "😊", text: "今天心情不错" },
  { word: "love", emoji: "💙", text: "好喜欢你" },
  { word: "shy", emoji: "😳", text: "别、别盯着看…" },
  { word: "surprised", emoji: "😮", text: "诶？！" },
  { word: "excited", emoji: "🤩", text: "有点期待" },
  { word: "sleepy", emoji: "😴", text: "看得有点困了…" },
  { word: "confused", emoji: "🤔", text: "这段代码好难懂" },
  { word: "thinking", emoji: "💭", text: "让我想想…" },
  { word: "smug", emoji: "😏", text: "我可是很厉害的" },
];

/** 点她一下的反应 */
const POKES: PetAction[] = [
  { word: "surprised", emoji: "😮", text: "呀！被戳到了" },
  { word: "happy", emoji: "😊", text: "嘿嘿" },
  { word: "wink", emoji: "😉", text: "偷偷眨个眼" },
  { word: "shy", emoji: "😳", text: "别戳啦…" },
];

const IDLE_MOOD = { emoji: "🫧", text: "发呆中…" };

/* ---------- 挂载 ---------- */

const rootElement = ref<HTMLElement | null>(null);
const { setPetMood, setPetBubble } = usePetMood();
const { singing, toggleSing, disposeSinging } = usePetSing();

/**
 * 舞台上的光标：一碗米饭。
 *
 * 素材包里没有食物类的图（tex/ feat/ 全是她的身体部件与表情，thumbs/ 是八套配色的头像），
 * 所以这里画一个内联 SVG —— 不往素材目录里塞东西，换素材包也不受影响。
 * 想换成自己的图：把 rice.png 丢进 public/pet/，再把 RICE_CURSOR 换成
 * `url("${packUrl("rice.png")}") 16 16, pointer` 即可。
 */
const RICE_SVG = [
  // 画稿按 32×32 的格子画，再放大到 48px 输出 —— 32px 在侧栏里看着太小了
  '<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 32 32">',
  '<g stroke="#24304a" stroke-width="1.2" stroke-linejoin="round" stroke-linecap="round">',
  // 米饭：三团隆起画出蓬松的边缘，再补几道米粒的沟
  '<path d="M8.7 17.5Q7.7 13.6 10.4 11.7Q10.5 8.9 13.5 8.5Q16 7.1 18.5 8.5Q21.5 8.9 21.6 11.7Q24.3 13.6 23.3 17.5Z"',
  ' fill="#fffdf7"/>',
  '<path d="M12.2 12.4c.5 1 .5 2 0 3M15.9 10.6c.5 1.2.5 2.4 0 3.6M19.6 12.4c.5 1 .5 2 0 3"',
  ' fill="none" stroke="#d3daea" stroke-width=".85"/>',
  // 碗
  '<path d="M4.6 17.4h22.8c-.9 5.4-5.3 9-11.4 9s-10.5-3.6-11.4-9Z" fill="#4D6BFE"/>',
  // 碗口的内圈，让白米饭和蓝碗之间有层次
  '<ellipse cx="16" cy="17.4" rx="11.4" ry="1.9" fill="#93a6ff"/>',
  '</g></svg>',
].join("");

// 热点落在碗口正中（画稿的 16,16 放大一倍后是 24,24），点击位置不会偏
const RICE_CURSOR = `url("data:image/svg+xml,${encodeURIComponent(RICE_SVG)}") 24 24, pointer`;

/**
 * 整块舞台都是米饭光标：这里是她的活动范围（她会朝光标走过来），
 * 所以不用逐帧判断"光标是否正好压在她身上"—— 那个判定会随着她走路、转身、
 * 起跳不停变化，光标便在她身边闪来闪去。写成固定的 CSS，进出舞台由浏览器自己管。
 */
const stageStyle = { cursor: RICE_CURSOR };

/** 舞台尺寸，kit 每帧都通过 host.bounds() 读它 */
const bounds: PetBounds = { W: 0, H: 0, floorY: 0, S: 0.4 };

let body: PetBody | null = null;
let frameHandle = 0;
let lastFrameAt = 0;
let actionTimer = 0;
let idleTimer = 0;
let clickTimer = 0;
let walkId = 0;
let lastPointerAt = 0;
/** 上一次成功下发走位的目标；NaN 表示还没有过（用 -1 当哨兵会和"光标在最左侧"的目标撞上） */
let lastWalkTarget = Number.NaN;
/** 上一次写给气泡的锚点，只有真的动了才写，免得每帧都触发 PetMood 重渲染 */
let lastBubbleX = Number.NaN;
let lastBubbleBottom = Number.NaN;
let schemeIndex = 0;
let outfit: PetOutfit = OUTFITS[0];
let resizeObserver: ResizeObserver | null = null;
let viewObserver: IntersectionObserver | null = null;
/** 侧栏滚出视野时不再推进动画：桌宠是装饰，没必要让她在看不见的地方一直跑 WebGL */
let inView = true;

/** public/pet 是运行期目录：用 document.baseURI 拼，挂在子路径下也能取到 */
const packUrl = (path: string) => new URL(`pet/${path}`, document.baseURI).href;

const measure = () => {
  const element = rootElement.value;
  if (!element) return;
  const { width, height } = element.getBoundingClientRect();
  bounds.W = width || 1;
  bounds.H = height || 1;
  // 脚踩在盒子底部往上一点，留出影子
  bounds.floorY = bounds.H * 0.96;
  // 贴满盒子：她的静息高度约 244 rig 单位，再留一点给跳跃和头发
  bounds.S = Math.min(bounds.H * 0.95, bounds.W * 0.95) / 256;
};

function pick<T>(list: readonly T[]): T {
  return list[Math.floor(Math.random() * list.length)];
}

const showMood = (action: PetAction) => {
  setPetMood({
    emoji: action.emoji,
    text: action.text,
    schemeId: SCHEMES[schemeIndex].id,
    schemeName: SCHEMES[schemeIndex].name,
    outfitId: outfit.id,
    outfitName: outfit.name,
  });
  window.clearTimeout(idleTimer);
  idleTimer = window.setTimeout(() => setPetMood(IDLE_MOOD), 5200);
};

const playAction = (action: PetAction = pick(ACTIONS)) => {
  body?.do(action.word);
  showMood(action);
};

/** 动作之间留 5～10 秒，像在发呆里偶尔动一下，而不是表演 */
const scheduleAction = () => {
  window.clearTimeout(actionTimer);
  actionTimer = window.setTimeout(
    () => {
      // 唱歌时不插随机动作：气泡里正显示歌词，她再乱动反而打断
      if (inView && !document.hidden && !isBusy() && !singing.value) playAction();
      scheduleAction();
    },
    5000 + Math.random() * 5000
  );
};

const applyScheme = () => {
  const scheme = SCHEMES[schemeIndex];
  // 同一套服装下是交叉淡入；换服装会重建骨架，是硬切
  void body?.setScheme(`${scheme.id}-${outfit.id}`, { fade: 0.45 });
  setPetMood({ schemeId: scheme.id, schemeName: scheme.name, outfitId: outfit.id, outfitName: outfit.name });
};

/** 双击随机换一套配色：抽到当前这套会看不出任何变化，所以先把当前这套排除掉 */
const onRandomScheme = () => {
  // 双击里已经点了两下，取消待处理的"戳一下"
  window.clearTimeout(clickTimer);
  const otherSchemes = SCHEMES.map((_, index) => index).filter((index) => index !== schemeIndex);
  schemeIndex = pick(otherSchemes);
  applyScheme();
  showMood({ word: "", emoji: "🎨", text: `换成${SCHEMES[schemeIndex].name}～` });
};

const onPoke = () => {
  if (!body) return;
  void toggleSing().then((nowSinging) => {
    if (!body) return;
    // 开唱时让她跟着跳两下，停的时候给个"唱完了"的小反应
    if (nowSinging) body.do("dance");
    else playAction(pick(POKES));
  });
};

/**
 * 某个屏幕坐标是不是落在她身上（判断"戳她一下"，而不是点到舞台空地）。
 *
 * 不能只信 layout().box：那个框是按绘制的缩放算出来的，她转身时 faceVis 会从 1 穿过 0 再到 -1，
 * 中间那十几帧横向缩放接近零 —— 框会缩成一条线，点在身上也会被当成点空地。
 * 所以横向用脚下位置 ± 一个固定半宽（她连头发带尾巴大约 200 个 rig 单位宽），
 * 纵向仍用 kit 的框（蹲下、跳起来时它会跟着走），两边取并集。
 */
const isOnPetAt = (clientX: number, clientY: number) => {
  if (!body || !rootElement.value) return false;
  const rect = rootElement.value.getBoundingClientRect();
  const x = clientX - rect.left;
  const y = clientY - rect.top;
  const layout = body.layout();
  const box = layout.box;

  const halfWidth = 100 * bounds.S;
  const left = Math.min(box.x, layout.x - halfWidth) - 4;
  const right = Math.max(box.x + box.w, layout.x + halfWidth) + 4;
  const top = box.y - 4;
  // 鞋底和影子会超出 kit 给的框一点，底边放宽到地面
  const bottom = Math.max(box.y + box.h, bounds.floorY) + 4;
  return x >= left && x <= right && y >= top && y <= bottom;
};

/** 单击先等 240ms：是双击就不戳，否则每次换配色都先被戳一下 */
const onClick = (event: MouseEvent) => {
  window.clearTimeout(clickTimer);
  if (!isOnPetAt(event.clientX, event.clientY)) return;
  clickTimer = window.setTimeout(onPoke, 240);
};

const onPointerMove = (event: PointerEvent) => {
  walkTowardPointer(event.clientX);
};

// 滑进来的那一下立刻反应：从右边进来就往右走，从左边进来就往左走
const onPointerEnter = (event: PointerEvent) => {
  walkTowardPointer(event.clientX, true);
};

const step = (now: number) => {
  // 切走标签页、或侧栏滚出视野再回来时 now 会跳很大一段，截到 50ms 免得她瞬移
  const dt = lastFrameAt ? Math.min(0.05, Math.max(0, (now - lastFrameAt) / 1000)) : 1 / 60;
  lastFrameAt = now;
  if (inView) body?.step(dt);
  syncBubble();
  frameHandle = requestAnimationFrame(step);
};

/**
 * 把她头顶的位置写给 PetMood.vue 画气泡。
 *
 * 用 layout().bubble 而不是自己拿脚的位置加一个身高：那个点来自素材包的骨架，
 * 她蹲下、跳起、被拎起来时会一起动，气泡就始终贴在头上，而不是挂在半空。
 * 位置保留一位小数就够，避免每帧都写一次 reactive 触发重渲染。
 */
const syncBubble = () => {
  if (!body) return;
  const head = body.layout().bubble;
  // 卡片底边离头顶留 10px，尾巴那截尖角（16px 方块转 45° 后约 11px）正好指着她
  const bubbleBottom = Math.round((bounds.H - head.y + 10) * 2) / 2;
  const bubbleX = Math.round(head.x * 2) / 2;
  if (bubbleX === lastBubbleX && bubbleBottom === lastBubbleBottom) return;
  lastBubbleX = bubbleX;
  lastBubbleBottom = bubbleBottom;
  setPetBubble({ x: bubbleX, bottom: bubbleBottom, visible: true });
};

/** 她是不是正在做需要连贯的事：走路、被拎、腾空时不插随机动作 */
const isBusy = () => {
  if (!body) return true;
  const layout = body.layout();
  return layout.busy || layout.moving || layout.mode === "walk" || layout.mode === "run";
};

/**
 * 朝光标走过去（只在她的舞台上生效）。之所以不按整页宽度映射 —— 那样光标停在侧栏时会被算成"最左边"，
 * 她会朝反方向跑掉。这里用舞台内的坐标：光标在哪一侧，她就往哪一侧走。
 */
const walkTowardPointer = (clientX: number, immediate = false) => {
  if (!body || !inView || !rootElement.value) return;
  const now = performance.now();
  // 移动时按 220ms 节流；kit 的 walk 每次会重置起步缓动，太密下发反而走不动
  if (!immediate && now - lastPointerAt < 220) return;
  lastPointerAt = now;

  const rect = rootElement.value.getBoundingClientRect();
  const current = body.layout().x;
  let target = Math.min(bounds.W, Math.max(0, clientX - rect.left));

  // 光标正好落在她身上（或旁边很近）时给一个小位移，不然"滑进来却不动"
  if (Math.abs(target - current) < 50) {
    const direction = target >= current ? 1 : -1;
    target = Math.min(bounds.W, Math.max(0, current + direction * 90));
  }
  if (Math.abs(target - lastWalkTarget) < 24) return;

  // 要跨越大半个舞台就小跑，近距离慢慢挪
  const accepted = body.walk(target, Math.abs(target - current) > bounds.W * 0.55, (walkId += 1));
  // 她正在落地、被拎着或做别的连贯动作时 kit 会拒掉这次走位：不记账，下一次移动再试一遍
  if (accepted) lastWalkTarget = target;
};

/** 加载素材包并造 body；贴图是一次性整批拉的，一张失败整只就没了，所以失败重试两次 */
const mountBody = async (): Promise<PetBody> => {
  const [kit, pack] = (await Promise.all([
    import(/* @vite-ignore */ packUrl("kit/body.js")),
    import(/* @vite-ignore */ packUrl("whale/figure.js")),
  ])) as [PetKitModule, WhaleFigureModule];

  const host: PetHost = {
    root: rootElement.value as HTMLElement,
    bounds: () => bounds,
    emit: () => {},
    sound: () => {},
    start: { x: bounds.W / 2, facing: 1, enter: "drop" },
  };

  let lastError: unknown;
  for (let attempt = 1; attempt <= 3; attempt += 1) {
    try {
      return await pack.createWhaleBody(new URL("pet/whale/", document.baseURI), {
        kit,
        host,
        scheme: `${SCHEMES[schemeIndex].id}-${outfit.id}`,
      });
    } catch (error) {
      lastError = error;
      if (attempt < 3) await new Promise((resolve) => window.setTimeout(resolve, 300));
    }
  }
  throw lastError;
};

onMounted(async () => {
  // 每次加载随机挑一套衣服：女仆装 / 冬装
  outfit = pick(OUTFITS);
  measure();

  const element = rootElement.value;
  if (!element) return;

  const observer = new ResizeObserver(() => {
    measure();
    body?.resize();
  });
  observer.observe(element);
  resizeObserver = observer;

  const view = new IntersectionObserver(([entry]) => {
    inView = entry.isIntersecting;
    // 回来时从头计时，否则第一帧的 dt 会让动作猛跳一下
    lastFrameAt = 0;
  });
  view.observe(element);
  viewObserver = view;

  try {
    body = await mountBody();
  } catch (error) {
    // 素材没加载出来就当没有桌宠：不抛错、不影响侧栏其它内容
    console.warn("[pet] 桌宠素材加载失败：", error);
    setPetMood({ emoji: "💤", text: "她走神了…" });
    return;
  }

  // 动作与移动都由这个文件驱动，关掉 kit 自己的自由漫游
  body.set({ roam: "off" });
  setPetMood({
    emoji: "👋",
    text: "我回来啦～",
    schemeId: SCHEMES[schemeIndex].id,
    schemeName: SCHEMES[schemeIndex].name,
    outfitId: outfit.id,
    outfitName: outfit.name,
  });

  // 侧栏滚出视野时 step 会空转（inView=false），回到视野再继续
  lastFrameAt = 0;
  // 先摆一次气泡，免得第一帧之前它停在舞台左下角闪一下
  syncBubble();
  frameHandle = requestAnimationFrame(step);
  // 问候语几秒后回到发呆；走 idleTimer，免得随机动作刚好在这之前发生、被这个定时器提前盖掉
  idleTimer = window.setTimeout(() => setPetMood(IDLE_MOOD), 4200);
  scheduleAction();
});

onBeforeUnmount(() => {
  cancelAnimationFrame(frameHandle);
  window.clearTimeout(actionTimer);
  window.clearTimeout(idleTimer);
  window.clearTimeout(clickTimer);
  resizeObserver?.disconnect();
  viewObserver?.disconnect();
  disposeSinging();
  body?.dispose();
  body = null;
  // 状态是模块级单例，不回收的话下次挂载前气泡会停在上一页的位置
  lastBubbleX = Number.NaN;
  lastBubbleBottom = Number.NaN;
  setPetBubble({ visible: false });
});
</script>
