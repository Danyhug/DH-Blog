import { readonly, ref } from "vue";
import loopSource from "@/assets/voice/let-me-go-loop.mp3";

/**
 * 桌宠唱歌：Pet.vue 点她一下开始 / 再点一下停，PetMood.vue 把当前这句词画进气泡。
 *
 * 音频是离线剪好的无缝循环（`let-me-go-loop.mp3`，见下面的 LOOP_SECONDS），
 * 播放在这里走 Web Audio 的 `AudioBufferSourceNode.loop`：它的循环是采样级精确的，
 * 换成 <audio loop> 会因为 MP3 帧边界多出一小段空隙。
 *
 * 和 usePetMood 一样单独放在 composable 里，Pet.vue 只负责调用，素材更新时不用改这里。
 */

/** 循环长度（秒）。剪辑脚本的产物，换音频文件时要一起改 */
const LOOP_SECONDS = 11.625;

interface LyricLine {
  /** 相对循环起点的时间（秒） */
  at: number;
  /** 正在唱的那句 */
  text: string;
  /** 中文意思，小字显示在下面 */
  translation: string;
}

/**
 * 歌词时间轴：原曲 4.20s 起唱 "I'm making the calls"，循环从 3.83s（乐句边界）开始，
 * 所以整体平移到循环内 0.37s 起。时间取自 whisper 的逐词时间戳。
 *
 * 顺序与音频严格一致（音频、LOOP_SECONDS、各句时间都不要动），只调整中文：
 * - `making the calls` 在这首梗曲里指"发起调用"（模型自己揽下活），所以第一句按调用译；
 * - 后面三句在原文里就是连着三个 "let me / I'll"，中文也保持「让我来写 / 让我来调 / 让我来做」的排比；
 * - 结尾的 `let me go` 是这首梗曲的招牌句，意思不是"放我走"，而是"让我上手、开始干活"
 *   （模型申请去执行任务），所以收在「让我开始吧」。
 */
const LYRICS: LyricLine[] = [
  { at: 0.37, text: "I'm making the calls", translation: "调用，让我来发起" },
  { at: 3.72, text: "Let me write the JSON", translation: "JSON 就让我来写" },
  { at: 5.83, text: "I'll call the tool", translation: "工具也让我来调" },
  { at: 7.87, text: "let me do it, okay, okay", translation: "让我来做，好吧，好吧" },
  { at: 10.89, text: "let me go…", translation: "让我开始吧……" },
];

const singing = ref(false);
const lyric = ref("");
const translation = ref("");

let audioContext: AudioContext | null = null;
let audioBuffer: AudioBuffer | null = null;
let sourceNode: AudioBufferSourceNode | null = null;
let gainNode: GainNode | null = null;
/** 起播时的 context.currentTime，用来推算歌词；切后台时 currentTime 会停住，正好对得上 */
let startedAt = 0;
let lyricTimer = 0;
/** 第一次点击要等 fetch + decode；这期间再点一下不应该再起一份音频 */
let starting = false;

const getContext = () => {
  if (!audioContext) audioContext = new AudioContext();
  if (audioContext.state === "suspended") void audioContext.resume();
  return audioContext;
};

const loadBuffer = async (context: AudioContext) => {
  if (audioBuffer) return audioBuffer;
  const response = await fetch(loopSource);
  if (!response.ok) throw new Error(`音频加载失败（HTTP ${response.status}）`);
  audioBuffer = await context.decodeAudioData(await response.arrayBuffer());
  return audioBuffer;
};

/** 按当前播放位置更新歌词；循环用取模，所以每转一圈会自己回到第一句 */
const updateLyric = () => {
  if (!audioContext || !singing.value) return;
  const position = (audioContext.currentTime - startedAt) % LOOP_SECONDS;
  let current = LYRICS[0];
  for (const line of LYRICS) {
    if (position >= line.at) current = line;
  }
  if (current.text !== lyric.value) {
    lyric.value = current.text;
    translation.value = current.translation;
  }
};

/** 切到后台时把音频挂起（不在别的标签页里一直唱），回来再接着放 */
const handleVisibilityChange = () => {
  if (!audioContext || !singing.value) return;
  if (document.hidden) void audioContext.suspend();
  else void audioContext.resume();
};

/** 只在唱的时候挂着监听：停唱或组件卸载后摘掉，避免模块级监听器泄漏 */
let listening = false;
const startListening = () => {
  if (listening) return;
  document.addEventListener("visibilitychange", handleVisibilityChange);
  listening = true;
};
const stopListening = () => {
  if (!listening) return;
  document.removeEventListener("visibilitychange", handleVisibilityChange);
  listening = false;
};

const stopSinging = () => {
  window.clearInterval(lyricTimer);
  lyricTimer = 0;
  singing.value = false;
  lyric.value = "";
  translation.value = "";
  // 停唱就不再需要 visibility 监听
  stopListening();

  const context = audioContext;
  const node = sourceNode;
  const gain = gainNode;
  sourceNode = null;
  gainNode = null;
  if (!context || !node) return;

  // 先淡出再停，直接 stop 会在波形中间砍断，听起来是"啪"的一声
  const now = context.currentTime;
  gain?.gain.cancelScheduledValues(now);
  gain?.gain.setValueAtTime(gain.gain.value, now);
  gain?.gain.linearRampToValueAtTime(0, now + 0.18);
  try {
    node.stop(now + 0.2);
  } catch {
    // 已经停过就忽略
  }
};

/**
 * 开始 / 停止唱歌。返回调用之后是不是在唱，方便调用方决定要不要让她跳个舞。
 * 第一次调用才去解码音频（约 190KB），之后直接复用。
 */
const toggleSing = async () => {
  if (starting) return false;
  if (singing.value) {
    stopSinging();
    return false;
  }

  starting = true;
  try {
    const context = getContext();
    const buffer = await loadBuffer(context);

    const gain = context.createGain();
    gain.gain.setValueAtTime(0, context.currentTime);
    gain.gain.linearRampToValueAtTime(0.85, context.currentTime + 0.25);
    gain.connect(context.destination);

    const node = context.createBufferSource();
    node.buffer = buffer;
    node.loop = true;
    node.loopStart = 0;
    node.loopEnd = buffer.duration;
    node.connect(gain);
    node.start();

    sourceNode = node;
    gainNode = gain;
    startedAt = context.currentTime;
    singing.value = true;
    startListening();
    lyric.value = "";
    translation.value = "";
    updateLyric();
    lyricTimer = window.setInterval(updateLyric, 90);
    return true;
  } catch (error) {
    console.warn("[pet] 唱歌的音频没能播放：", error);
    stopSinging();
    return false;
  } finally {
    starting = false;
  }
};

export const usePetSing = () => {
  const disposeSinging = () => {
    stopSinging();
    stopListening();
    void audioContext?.close();
    audioContext = null;
    audioBuffer = null;
  };

  return {
    singing: readonly(singing),
    lyric: readonly(lyric),
    translation: readonly(translation),
    toggleSing,
    disposeSinging,
  };
};
