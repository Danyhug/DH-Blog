import { getArticleCategoryList, getArticleTagList, getSiteConfig } from "@/api/user";
import { ArticleModel } from "@/types/ArticleModel";
import { Category } from "@/types/Category";
import { Tag } from "@/types/Tag";
import { defineStore } from "pinia";
import { computed, reactive, ref, watch } from "vue";
import { MdInit } from "@/types/MdEditor";
import { Article } from "@/types/Article";
import { Page } from "@/types/Page";
import { SiteConfig } from "@/types/SystemConfig";
import {
  createPlaylist,
  deletePlaylist,
  getMusicLibrary,
  listPlaylists,
  mediaStreamUrl,
  saveProgress,
  trackCoverUrl,
  updatePlaylist,
  type MissingMedia,
  type Playlist,
  type Track,
} from "@/api/media";
import { shuffled } from "@/views/frontend/theater/utils/format";
import { rememberDuration } from "@/views/frontend/theater/utils/durationProbe";
import { notify } from "@/utils/notification";

// 站点公开配置：后台「站点设置」的内容，前台展示用
export const useSiteStore = defineStore("site", () => {
  const site = reactive<SiteConfig>({
    blog_title: "DH-Blog",
    signature: "",
    avatar: "",
    github_link: "",
    bilibili_link: "",
    open_comment: true,
  });
  const loaded = ref(false);

  const loadSite = async () => {
    if (loaded.value) return;
    try {
      const data = await getSiteConfig();
      // 后端返回空串时保留默认值，避免页面出现空标题
      Object.assign(site, {
        ...data,
        blog_title: data.blog_title || site.blog_title,
      });
    } catch {
      // 站点配置拉取失败时沿用默认值，不阻塞页面渲染
    } finally {
      loaded.value = true;
    }
  };

  return { site, loaded, loadSite };
});

export const useSystemStore = defineStore("system", () => {
  const mdEditorInit = reactive<MdInit>({
    codeFoldable: true,
    editorId: "dh-editor",
    previewTheme: "cyanosis",
    theme: "light",
  });

  return {
    mdEditorInit,
  };
});

export const useAdminStore = defineStore("admin", () => {
  const tags = reactive<Tag[]>([]);
  const categories = reactive<Category[]>([]);
  const online = ref(0);

  const getCategories = async () => {
    const data = await getArticleCategoryList();
    categories.splice(0, categories.length, ...data);
  };

  // 获取标签列表
  const getTags = async () => {
    const data = await getArticleTagList();
    tags.splice(0, tags.length, ...data);
  };

  return {
    tags,
    categories,
    getCategories,
    getTags,
    online,
  };
});

export const useUserStore = defineStore("user", () => {
  const homeShowComponent = ref("home");
  const commentKey = ref(true)
  const isBan = ref(false);

  // 首页上方展示内容（文章详情上面）
  interface HomeHeaderInfo {
    id: number;
    title: string;
    created: string;
    wordNum: number;
    tags: any;
    thumbnailUrl: string;
    timConSum: string;
  }
  const homeHeaderInfo = reactive<HomeHeaderInfo>({
    id: -1,
    title: "",
    created: "",
    // 总字数
    wordNum: 0,
    // 阅读时长
    timConSum: "0",
    thumbnailUrl: "",
    tags: [],
  });

  // 文章状态控制
  const readingStorageKey = 'dh-blog:reading';
  let readingPreferences = { isDarkMode: false, fontSize: 16 };
  try {
    const saved = JSON.parse(localStorage.getItem(readingStorageKey) || 'null');
    if (saved && typeof saved === 'object') {
      readingPreferences = {
        isDarkMode: saved.isDarkMode === true,
        fontSize: Number.isInteger(saved.fontSize) && saved.fontSize >= 14 && saved.fontSize <= 24
          ? saved.fontSize : 16,
      };
    }
  } catch {
    // Reading controls still work when storage is unavailable or malformed.
  }
  const aritcleModel = reactive<ArticleModel>({
    ...readingPreferences,
    isFullPreview: false,
  });

  watch(() => [aritcleModel.isDarkMode, aritcleModel.fontSize], () => {
    try {
      localStorage.setItem(readingStorageKey, JSON.stringify({
        isDarkMode: aritcleModel.isDarkMode,
        fontSize: aritcleModel.fontSize,
      }));
    } catch {
      // Persist only preferences, never the transient focus-reading state.
    }
  });

  // 首页文章列表
  const articleList = reactive<Article<Tag[]>[]>([]);
  const page = reactive<Page>({
    pageNum: 1,
    pageSize: 7,
    total: 0,
  });

  return {
    homeShowComponent,
    homeHeaderInfo,
    aritcleModel,
    articleList,
    page,
    commentKey,
    isBan
  };
});


// ========== 个人影院 · 音乐库 ==========
// 曲库与歌单放在 store 里，是因为音乐页的各个视图和全局播放条要共享同一份曲目对象：
// 播放器探测到的时长、播放次数写回对象后，列表里立刻能看到。
export const useMusicStore = defineStore("music", () => {
  const tracks = ref<Track[]>([]);
  const playlists = ref<Playlist[]>([]);
  const scoped = ref(false);
  const missing = ref<MissingMedia>({ count: 0, samples: [] });
  const loaded = ref(false);
  const loading = ref(false);

  const trackById = computed(() => new Map(tracks.value.map((track) => [track.id, track])));

  const load = async (force = false) => {
    if (loading.value || (loaded.value && !force)) return;
    loading.value = true;
    try {
      const [library, lists] = await Promise.all([getMusicLibrary(), listPlaylists()]);
      tracks.value = library.tracks;
      scoped.value = library.scoped;
      missing.value = library.missing ?? { count: 0, samples: [] };
      playlists.value = lists;
      loaded.value = true;
    } finally {
      loading.value = false;
    }
  };

  const playlistTracks = (playlist: Playlist): Track[] =>
    playlist.track_ids.map((id) => trackById.value.get(id)).filter((track): track is Track => !!track);

  const replacePlaylist = (updated: Playlist) => {
    const index = playlists.value.findIndex((item) => item.id === updated.id);
    if (index >= 0) playlists.value.splice(index, 1, updated);
    else playlists.value.push(updated);
    return updated;
  };

  const addPlaylist = async (name: string, trackIds: string[] = []) =>
    replacePlaylist(await createPlaylist(name, trackIds));

  const renamePlaylist = async (playlist: Playlist, name: string, description?: string) =>
    replacePlaylist(await updatePlaylist(playlist.id, { name, description }));

  const setPlaylistTracks = async (playlist: Playlist, trackIds: string[]) =>
    replacePlaylist(await updatePlaylist(playlist.id, { track_ids: trackIds }));

  const addToPlaylist = async (playlist: Playlist, trackIds: string[]) => {
    const merged = [...playlist.track_ids, ...trackIds.filter((id) => !playlist.track_ids.includes(id))];
    return setPlaylistTracks(playlist, merged);
  };

  const removePlaylist = async (playlist: Playlist) => {
    await deletePlaylist(playlist.id);
    playlists.value = playlists.value.filter((item) => item.id !== playlist.id);
  };

  const markPlayed = (id: string) => {
    const track = trackById.value.get(id);
    if (!track) return;
    track.play_count = (track.play_count || 0) + 1;
    const now = new Date();
    const pad = (value: number) => String(value).padStart(2, "0");
    track.last_played_at = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`;
  };

  return {
    tracks,
    playlists,
    scoped,
    missing,
    loaded,
    loading,
    trackById,
    load,
    playlistTracks,
    addPlaylist,
    renamePlaylist,
    setPlaylistTracks,
    addToPlaylist,
    removePlaylist,
    markPlayed,
  };
});

type RepeatMode = "off" | "all" | "one";

const volumeStorageKey = "dh-blog:music-volume";

function readVolume(): number {
  try {
    const value = Number(localStorage.getItem(volumeStorageKey));
    return Number.isFinite(value) && value >= 0 && value <= 1 && localStorage.getItem(volumeStorageKey) !== null ? value : 0.8;
  } catch {
    return 0.8;
  }
}

// ========== 个人影院 · 音乐播放器 ==========
// 全局唯一的 <audio>，挂在 store 里而不是组件里：切换音乐页的视图、甚至去看影视页时，
// 播放都不会中断，底部播放条和全屏「正在播放」只是它的两种外观。
export const useMusicPlayerStore = defineStore("musicPlayer", () => {
  /** 实际播放顺序（随机模式下已打乱） */
  const queue = ref<Track[]>([]);
  /** 打乱前的原始顺序，关闭随机时据此还原 */
  let ordered: Track[] = [];
  const index = ref(-1);
  const playing = ref(false);
  const buffering = ref(false);
  const currentTime = ref(0);
  const duration = ref(0);
  const buffered = ref(0);
  const volume = ref(readVolume());
  const muted = ref(false);
  const shuffle = ref(false);
  const repeat = ref<RepeatMode>("off");
  /** 全屏「正在播放」是否展开，以及展开后右侧显示歌词还是待播清单 */
  const expanded = ref(false);
  const panel = ref<"lyrics" | "queue">("lyrics");

  const current = computed<Track | null>(() => queue.value[index.value] ?? null);
  const upNext = computed(() => queue.value.slice(index.value + 1));

  let audio: HTMLAudioElement | null = null;
  // 同一首歌只在真正开始出声时记一次播放
  let startedId = "";

  const element = (): HTMLAudioElement => {
    if (audio) return audio;
    const el = new Audio();
    el.preload = "auto";
    el.volume = volume.value;
    el.addEventListener("timeupdate", () => (currentTime.value = el.currentTime));
    el.addEventListener("durationchange", () => {
      if (!Number.isFinite(el.duration)) return;
      duration.value = el.duration;
      const track = current.value;
      if (track && !track.duration) {
        track.duration = el.duration;
        rememberDuration(track.id, el.duration);
      }
    });
    el.addEventListener("progress", () => {
      buffered.value = el.buffered.length ? el.buffered.end(el.buffered.length - 1) : 0;
    });
    el.addEventListener("play", () => (playing.value = true));
    el.addEventListener("pause", () => (playing.value = false));
    el.addEventListener("waiting", () => (buffering.value = true));
    el.addEventListener("playing", () => {
      buffering.value = false;
      const track = current.value;
      if (track && startedId !== track.id) {
        startedId = track.id;
        saveProgress(track.id, 0, track.duration || 0, true).catch(() => {});
        useMusicStore().markPlayed(track.id);
      }
    });
    el.addEventListener("canplay", () => (buffering.value = false));
    el.addEventListener("ended", onEnded);
    el.addEventListener("error", () => {
      if (!el.getAttribute("src")) return;
      const track = current.value;
      buffering.value = false;
      notify.error(`无法播放「${track?.title || "未知曲目"}」，浏览器可能不支持该音频格式`);
      if (index.value < queue.value.length - 1) setTimeout(() => next(true), 800);
    });
    audio = el;
    return el;
  };

  const updateMediaSession = () => {
    const track = current.value;
    if (!("mediaSession" in navigator) || !track) return;
    const cover = trackCoverUrl(track);
    navigator.mediaSession.metadata = new MediaMetadata({
      title: track.title,
      artist: track.artist,
      album: track.album,
      artwork: cover ? [{ src: new URL(cover, location.href).href, sizes: "512x512" }] : [],
    });
    navigator.mediaSession.setActionHandler("play", () => element().play());
    navigator.mediaSession.setActionHandler("pause", () => element().pause());
    navigator.mediaSession.setActionHandler("previoustrack", () => prev());
    navigator.mediaSession.setActionHandler("nexttrack", () => next());
    navigator.mediaSession.setActionHandler("seekto", (details) => {
      if (details.seekTime !== undefined) seek(details.seekTime);
    });
  };

  const loadCurrent = (autoplay = true) => {
    const el = element();
    const track = current.value;
    startedId = "";
    currentTime.value = 0;
    buffered.value = 0;
    if (!track) {
      el.pause();
      el.removeAttribute("src");
      el.load();
      duration.value = 0;
      return;
    }
    duration.value = track.duration || 0;
    el.src = mediaStreamUrl(track.id);
    if (autoplay) {
      buffering.value = true;
      el.play().catch(() => (buffering.value = false));
    }
    updateMediaSession();
  };

  /** 播放一组曲目。startIndex 指向列表中的起始曲目；shuffle=true 时从随机位置开始 */
  const playList = (tracks: Track[], startIndex = 0, options: { shuffle?: boolean } = {}) => {
    if (!tracks.length) return;
    ordered = tracks.slice();
    if (options.shuffle !== undefined) shuffle.value = options.shuffle;
    if (shuffle.value) {
      const first = options.shuffle ? Math.floor(Math.random() * tracks.length) : startIndex;
      queue.value = [tracks[first], ...shuffled(tracks.filter((_, i) => i !== first))];
      index.value = 0;
    } else {
      queue.value = ordered.slice();
      index.value = Math.min(Math.max(startIndex, 0), tracks.length - 1);
    }
    loadCurrent();
  };

  const toggle = () => {
    if (!current.value) return;
    const el = element();
    if (el.paused) el.play().catch(() => {});
    else el.pause();
  };

  const pause = () => audio?.pause();

  /** auto=true 表示由播放结束触发：队列放完就停，手动「下一首」则不做任何事 */
  const next = (auto = false) => {
    if (!queue.value.length) return;
    if (index.value < queue.value.length - 1) {
      index.value++;
    } else if (repeat.value === "all") {
      index.value = 0;
    } else {
      if (auto) {
        element().pause();
        seek(0);
      }
      return;
    }
    loadCurrent();
  };

  const prev = () => {
    // 与 Apple Music 一致：播放超过 3 秒时「上一首」先回到本曲开头
    if (currentTime.value > 3 || index.value <= 0) {
      seek(0);
      return;
    }
    index.value--;
    loadCurrent();
  };

  function onEnded() {
    const track = current.value;
    if (track) saveProgress(track.id, track.duration || duration.value, track.duration || duration.value).catch(() => {});
    if (repeat.value === "one") {
      seek(0);
      element().play().catch(() => {});
      return;
    }
    next(true);
  }

  const seek = (time: number) => {
    const el = element();
    if (!Number.isFinite(time)) return;
    el.currentTime = Math.max(0, Math.min(time, duration.value || time));
    currentTime.value = el.currentTime;
  };

  const setVolume = (value: number) => {
    volume.value = Math.max(0, Math.min(1, value));
    muted.value = volume.value === 0;
    const el = element();
    el.volume = volume.value;
    el.muted = muted.value;
    try {
      localStorage.setItem(volumeStorageKey, String(volume.value));
    } catch {
      // 音量记忆只是便利功能
    }
  };

  const toggleMute = () => {
    muted.value = !muted.value;
    element().muted = muted.value;
    if (!muted.value && volume.value === 0) setVolume(0.5);
  };

  const toggleShuffle = () => {
    shuffle.value = !shuffle.value;
    const track = current.value;
    if (!track) return;
    if (shuffle.value) {
      const rest = queue.value.filter((_, i) => i !== index.value);
      queue.value = [track, ...shuffled(rest)];
      index.value = 0;
    } else {
      queue.value = ordered.slice();
      index.value = Math.max(0, queue.value.findIndex((item) => item.id === track.id));
    }
  };

  const cycleRepeat = () => {
    repeat.value = repeat.value === "off" ? "all" : repeat.value === "all" ? "one" : "off";
  };

  /** 「下一首播放」：插到当前曲目后面 */
  const playNext = (tracks: Track[]) => {
    if (!current.value) return playList(tracks);
    queue.value.splice(index.value + 1, 0, ...tracks);
    const at = ordered.findIndex((item) => item.id === current.value!.id);
    ordered.splice(at + 1, 0, ...tracks);
  };

  /** 「添加到播放队列」：排到队尾 */
  const addToQueue = (tracks: Track[]) => {
    if (!current.value) return playList(tracks);
    queue.value.push(...tracks);
    ordered.push(...tracks);
  };

  const jumpTo = (position: number) => {
    if (position < 0 || position >= queue.value.length) return;
    index.value = position;
    loadCurrent();
  };

  const removeFromQueue = (position: number) => {
    if (position === index.value || position < 0 || position >= queue.value.length) return;
    const [removed] = queue.value.splice(position, 1);
    if (position < index.value) index.value--;
    const at = ordered.findIndex((item) => item.id === removed.id);
    if (at >= 0) ordered.splice(at, 1);
  };

  const clearUpNext = () => {
    const track = current.value;
    queue.value = track ? [track] : [];
    ordered = queue.value.slice();
    index.value = track ? 0 : -1;
  };

  return {
    queue,
    index,
    playing,
    buffering,
    currentTime,
    duration,
    buffered,
    volume,
    muted,
    shuffle,
    repeat,
    expanded,
    panel,
    current,
    upNext,
    playList,
    toggle,
    pause,
    next,
    prev,
    seek,
    setVolume,
    toggleMute,
    toggleShuffle,
    cycleRepeat,
    playNext,
    addToQueue,
    jumpTo,
    removeFromQueue,
    clearUpNext,
  };
});
