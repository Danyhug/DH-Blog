<template>
    <div class="knowledge-page flex flex-col h-screen bg-[var(--bg-color)] text-[var(--text-color)] overflow-hidden [font-family:[-apple-system,BlinkMacSystemFont,'Segoe_UI',Roboto,'Helvetica_Neue',Arial,sans-serif]]">
        <div id="particles-js" class="fixed top-0 left-0 w-full h-full z-0"></div>
        <main class="main-container flex grow p-10 gap-10 z-[1] relative h-[calc(100vh-80px)]">
            <section class="content-section animate-fade-in-up">
                <h2 class="section-title text-[2em] text-center text-[var(--text-color)] py-[25px] px-5 m-0 shrink-0 [text-shadow:none] border-b border-b-[var(--border-color)]">📂 文章分类</h2>
                <div class="grid-wrapper grow p-[25px] [scrollbar-gutter:stable]">
                    <div v-if="isLoading" class="loading-state text-center p-10 text-[var(--text-color)] text-[1.1em]">
                        <i class="fas fa-spinner fa-spin mr-[10px]"></i> 加载中...
                    </div>
                    <div v-else class="grid gap-5">
                        <!-- ✨ Staggering achieved via inline style -->
                        <a v-for="(category, index) in categories" :key="category.name" href="#" class="card animate-fade-in-up"
                            :style="{ animationDelay: 200 + index * 50 + 'ms' }" @click.prevent="openModal(category)">
                            <div class="card-content">
                                <span class="[transition:color_0.3s_ease-out]">{{ category.name }}</span>
                                <span class="count">{{ category.count }}篇</span>
                            </div>
                        </a>
                    </div>
                </div>
            </section>

            <!-- Tags Section -->
            <section class="content-section tag-section animate-fade-in-up" style="flex: 1">
                <h2 class="section-title text-[2em] text-center text-[var(--text-color)] py-[25px] px-5 m-0 shrink-0 [text-shadow:none] border-b border-b-[var(--border-color)]">🏷️ 热门标签</h2>
                <div class="grid-wrapper grow p-[25px] [scrollbar-gutter:stable]">
                    <div v-if="isLoading" class="loading-state text-center p-10 text-[var(--text-color)] text-[1.1em]">
                        <i class="fas fa-spinner fa-spin mr-[10px]"></i> 加载中...
                    </div>
                    <div v-else class="grid gap-5 grid-cols-[repeat(auto-fill,minmax(130px,1fr))]">
                        <!-- ✨ Staggering achieved via inline style -->
                        <a v-for="(tag, index) in tags" :key="tag.name" href="#" class="card tag-card animate-fade-in-up"
                            :style="{ animationDelay: 800 + index * 50 + 'ms' }" @click.prevent="openModal(tag)">
                            <div class="card-content">
                                <span class="[transition:color_0.3s_ease-out]">{{ tag.name }}</span>
                                <span class="badge">{{ tag.count }}</span>
                            </div>
                        </a>
                    </div>
                </div>
            </section>
        </main>

        <!-- Modal with Vue Transition -->
        <!-- ✨ Replaced GSAP with Vue's <Transition> component -->
        <Transition name="modal-fade">
            <div v-if="isModalVisible" class="modal-overlay fixed top-0 left-0 w-full h-full bg-[rgba(244,247,252,0.8)] z-[1000] flex justify-center items-center p-5" @click.self="closeModal">
                <Transition name="modal-zoom">
                    <div v-if="isModalVisible" class="modal-content bg-[var(--card-bg-color)] border border-[var(--border-color)] rounded-2xl py-[25px] px-[30px] w-full max-w-[700px] max-h-[85vh] flex flex-col relative shadow-[0_15px_50px_rgba(45,55,72,0.15)]">
                        <button class="modal-close-btn absolute top-[15px] right-[15px] bg-[#EDF2F7] [border:none] text-[var(--text-color)] w-9 h-9 rounded-[50%] text-[24px] leading-[36px] text-center cursor-pointer [transition:background_0.3s,transform_0.3s]" @click="closeModal">×</button>
                        <h3 class="modal-title text-[1.8em] text-[var(--text-color)] mx-0 mt-0 mb-5 pb-[15px] pr-10 border-b border-b-[var(--border-color)] [text-shadow:none]">{{ modalTitle }}</h3>
                        <ul class="article-list list-none p-0 pr-[15px] m-0 overflow-y-auto grow">
                            <li v-for="(article, index) in modalArticles" :key="index" @click="goToArticle(article)"
                                class="border-b border-b-[var(--border-color)] cursor-pointer py-[18px] pr-[10px] pl-5 text-[1.1em] [transition:all_0.3s_ease] relative">
                                <div>{{ article.title }}</div>
                                <div class="article-info text-[0.85em] text-[#666] mt-[6px]">
                                    <span class="mr-3"><i class="fas fa-eye mr-1"></i> {{ article.views }} 阅读</span>
                                    <span class="mr-3"><i class="fas fa-file-word mr-1"></i> {{ article.wordNum }} 字</span>
                                    <span class="mr-3"><i class="fas fa-calendar-alt mr-1"></i> {{ article.createTime }}</span>
                                </div>
                            </li>
                        </ul>
                    </div>
                </Transition>
            </div>
        </Transition>
    </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { getAllTaxonomies, getArticlesByTaxonomy } from '@/api/user';
import { useRouter } from 'vue-router';

// --- Reactive State Management ---
const allData = ref([]);
const isLoading = ref(false);
const isModalVisible = ref(false);
const modalTitle = ref('');
const modalArticles = ref([]);
const router = useRouter();

const categories = computed(() => allData.value.filter(item => item.type === 'category'));
const tags = computed(() => allData.value.filter(item => item.type === 'tag'));

// --- API Methods ---
const loadTaxonomies = async () => {
    isLoading.value = true;
    try {
        const data = await getAllTaxonomies();
        allData.value = data;
    } catch (error) {
        console.error('获取标签和分类失败:', error);
    } finally {
        isLoading.value = false;
    }
};

const openModal = async (item) => {
    modalTitle.value = `${item.name} - 文章列表`;
    modalArticles.value = [];
    
    try {
        const articles = await getArticlesByTaxonomy(item.name, item.type);
        modalArticles.value = articles.map(article => ({
            id: article.id,
            title: article.title,
            views: article.views,
            wordNum: article.wordNum,
            createTime: article.createTime
        }));
    } catch (error) {
        console.error('获取文章列表失败:', error);
        modalArticles.value = [{ id: 0, title: '加载文章失败，请稍后重试' }];
    }
    
    isModalVisible.value = true;
};

const closeModal = () => {
    isModalVisible.value = false;
};

const goToArticle = (article) => {
    if (article.id && article.id > 0) {
        const url = router.resolve({ name: 'ArticleInfo', params: { id: article.id } }).href;
        window.open(url, '_blank');
        closeModal();
    }
};

// --- Lifecycle Hook ---
onMounted(() => {
    // Initialize Particles.js (still needed)
    if (window.particlesJS) {
        window.particlesJS('particles-js', {
            "particles": { "number": { "value": 60, "density": { "enable": true, "value_area": 800 } }, "color": { "value": "#555555" }, "shape": { "type": "circle" }, "opacity": { "value": 0.4, "random": true }, "size": { "value": 3, "random": true }, "line_linked": { "enable": true, "distance": 150, "color": "#CCCCCC", "opacity": 0.4, "width": 1 }, "move": { "enable": true, "speed": 2, "direction": "none", "random": false, "straight": false, "out_mode": "out", "bounce": false } }, "interactivity": { "detect_on": "canvas", "events": { "onhover": { "enable": true, "mode": "repulse" }, "onclick": { "enable": true, "mode": "push" }, "resize": true }, "modes": { "repulse": { "distance": 100, "duration": 0.4 }, "push": { "particles_nb": 4 } } }, "retina_detect": true
        });
    }
    loadTaxonomies();
});
</script>

<style scoped>
/*
  Knowledge 页的样式大多是「页面骨架 + 弹窗」，已尽量内联到模板。
  下面只留工具类表达不了的，以及 §4.5 认可的组件类：

  1. 页面级 CSS 自定义属性（`--accent-color-*` / `--bg-color` / `--text-color` …）——
     被模板里的 `bg-[var(--bg-color)]` 等工具类与下面保留的规则共同引用。
  2. `.grid-wrapper` 的 overflow 三连：先 `hidden auto`，再写 `overflow-y: overlay`
     作渐进增强（`overlay` 非法时会回落到上一行的 `auto`）。这种「同属性两条声明、
     后者可能非法」的顺序语义，工具类表达不了（不写回退会掉成 `visible`）。
  3. 伪元素：`::-webkit-scrollbar*` 四组、`.article-list li::before`。
  4. 结构伪类与祖先 hover：`:first-of-type` / `:nth-of-type(2)` / `:last-child` / `:hover`。
  5. Vue `<Transition>` 运行时生成的 `.modal-fade-*` / `.modal-zoom-*`。
  6. 移动端覆盖块（无层级，自带 `!important`，正好维持原有的断点行为）。
  7. 组件类：`.card` / `.card-content` / `.count` / `.badge` 在 v-for 里重复渲染（§4.5）。

  ⚠️ 顺带删掉一批**死代码**：原 CSS 里的 `.section-categories` / `.section-tags`
  及其全部后代规则从未生效 —— 模板用的是裸 `.content-section` 与
  `.content-section.tag-section`，这两个类名在模板里不存在（全项目 grep 只在本文件的
  `<style>` 内命中），而 scoped 规则必须带 `[data-v-53a0e8e4]` 才能命中本组件元素。
  连带只包含这两者的 `@media (min-width: 901px)` 整块一并删除。

  `@keyframes fadeInUp` 已按 §4.1 挪进 tailwind.css（`--animate-fade-in-up`），
  模板改用 `animate-fade-in-up`：`:first-of-type` 那两条是无层级声明，
  照样覆盖工具类简写里的默认 `animation-delay: 0s`。
*/
.knowledge-page {
    --accent-color-1: hsl(180, 100%, 40%);
    --accent-color-2: hsl(280, 100%, 55%);
    --bg-color: #F4F7FC;
    --card-bg-color: #FFFFFF;
    --text-color: #2D3748;
    --text-color-light: #FFFFFF;
    --border-color: #E2E8F0;
    --shadow-color: rgba(45, 55, 72, 0.1);
    --accent-color-1-rgb: 0, 204, 204;
}

/* 顺序敏感：后一行非法时回落到前一行 */
.grid-wrapper {
    overflow-x: hidden;
    overflow-y: auto;
    overflow-y: overlay;
}

.grid-wrapper::-webkit-scrollbar {
    width: 8px;
}

.grid-wrapper::-webkit-scrollbar-track {
    background: transparent;
}

.grid-wrapper::-webkit-scrollbar-thumb {
    background: #CBD5E0;
    border-radius: 4px;
}

.grid-wrapper::-webkit-scrollbar-thumb:hover {
    background: #A0AEC0;
}

/* 两个 section 的入场错峰（Vue 拿不到这两条，只能留在 CSS） */
.content-section:first-of-type {
    animation-delay: 0.2s;
}

.content-section:nth-of-type(2) {
    animation-delay: 0.4s;
}

/* 组件类：v-for 里重复渲染的卡片（§4.5）。入场动画改由模板上的 animate-fade-in-up 承担 */
.card {
    border: 2px solid transparent;
    border-radius: 12px;
    padding: 0;
    text-decoration: none;
    position: relative;
    cursor: pointer;
    will-change: transform;
    transition: transform 0.3s ease-out, box-shadow 0.3s ease-out;
    background: var(--card-bg-color);
    box-shadow: 0 4px 15px var(--shadow-color);
    transform-origin: center;
}

.card-content {
    background: transparent;
    border-radius: 10px;
    padding: 20px;
    text-align: center;
    color: var(--text-color);
    font-size: 1.1em;
    font-weight: 500;
    display: flex;
    justify-content: center;
    align-items: center;
    position: relative;
    overflow: hidden;
}

.count {
    background: #EDF2F7;
    color: #718096;
    font-size: 0.8em;
    padding: 4px 10px;
    border-radius: 20px;
    margin-left: 15px;
    transition: color 0.3s ease-out, background-color 0.3s ease-out;
}

.badge {
    position: absolute;
    top: 0;
    right: 2px;
    background: linear-gradient(135deg, #ff6b6b, #ff8e8e);
    color: white;
    border-radius: 50%;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: bold;
    box-shadow: 0 2px 8px rgba(255, 107, 107, 0.3);
    z-index: 1;
}

.card:hover {
    transform: translateY(-4px) scale(1.05) rotate(2deg);
    box-shadow: 10px 10px 25px var(--shadow-color);
    border-radius: 12px;
}

/* Vue <Transition> 运行时生成的类，模板里静态写不出来 */
.modal-fade-enter-active,
.modal-fade-leave-active {
    transition: opacity 0.4s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
    opacity: 0;
}

.modal-zoom-enter-active {
    transition: all 0.4s cubic-bezier(0.215, 0.61, 0.355, 1);
    transition-delay: 0.1s;
}

.modal-zoom-leave-active {
    transition: all 0.3s ease-in;
}

.modal-zoom-enter-from,
.modal-zoom-leave-to {
    opacity: 0;
    transform: translateY(50px) scale(0.95);
}

.modal-close-btn:hover {
    background: #E2E8F0;
    transform: rotate(90deg);
}

.article-list::-webkit-scrollbar {
    width: 8px;
}

.article-list::-webkit-scrollbar-track {
    background: transparent;
}

.article-list::-webkit-scrollbar-thumb {
    background: #CBD5E0;
    border-radius: 4px;
}

.article-list::-webkit-scrollbar-thumb:hover {
    background: #A0AEC0;
}

.article-list li:last-child {
    border-bottom: none;
}

.article-list li:hover {
    background-color: #F7FAFC;
    color: var(--accent-color-1);
    padding-left: 25px;
}

.article-list li::before {
    content: '›';
    position: absolute;
    left: 8px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--accent-color-1);
    opacity: 0;
    transition: opacity 0.3s;
    font-weight: bold;
}

.article-list li:hover::before {
    opacity: 1;
}

/*
  只此一条：原文同时写了 `backdrop-filter: blur(5px)` 与 `-webkit-backdrop-filter: blur(5px)`，
  lightningcss 按目标浏览器把两者归并，最终产物**只剩 `-webkit-` 那行**；
  而实测当前 Chrome（154）不认 `-webkit-backdrop-filter`（隔离实验：只写它时边缘强度 6.01，
  与完全不写相同；只写无前缀时 0.47）——**这个模糊在 Chrome 下一直是失效的**，
  只有 Safari 之类认前缀的浏览器能看到。
  本方案只做样式迁移、不改观感，故原样保留。若要用 `backdrop-blur-[5px]` 打开它，
  请单开一个 commit（已登记在文档 §6.1）。
*/
.modal-overlay {
    -webkit-backdrop-filter: blur(5px);
}

@media (max-width: 900px) {
    .knowledge-page {
        height: auto;
        min-height: 100vh;
    }

    .main-container {
        flex-direction: column;
        height: auto;
        padding: 20px;
        gap: 20px;
    }

    .content-section {
        flex-basis: auto !important;
        height: 50vh;
    }

    .section-title {
        font-size: 1.5em;
        padding: 20px;
    }

    .grid-wrapper {
        padding: 20px;
    }

    .modal-content {
        padding: 20px;
    }

    .modal-title {
        font-size: 1.5em;
    }
}
</style>
