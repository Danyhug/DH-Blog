<template>
    <div class="knowledge-page [@media(max-width:900px)]:h-auto [@media(max-width:900px)]:min-h-screen flex flex-col h-screen bg-[var(--bg-color)] text-[var(--text-color)] overflow-hidden [font-family:[-apple-system,BlinkMacSystemFont,'Segoe_UI',Roboto,'Helvetica_Neue',Arial,sans-serif]]">
        <div id="particles-js" class="fixed top-0 left-0 w-full h-full z-0"></div>
        <main class="[@media(max-width:900px)]:flex-col [@media(max-width:900px)]:h-auto [@media(max-width:900px)]:p-5 [@media(max-width:900px)]:gap-5 flex grow p-10 gap-10 z-[1] relative h-[calc(100vh-80px)]">
            <section class="first-of-type:[animation-delay:0.2s] [&:nth-of-type(2)]:[animation-delay:0.4s] [@media(max-width:900px)]:basis-auto! [@media(max-width:900px)]:h-[50vh] animate-fade-in-up">
                <h2 class="[@media(max-width:900px)]:text-[1.5em] [@media(max-width:900px)]:p-5 text-[2em] text-center text-[var(--text-color)] py-[25px] px-5 m-0 shrink-0 [text-shadow:none] border-b border-b-[var(--border-color)]">📂 文章分类</h2>
                <div class="grid-wrapper [@media(max-width:900px)]:p-5 [&::-webkit-scrollbar]:w-2! [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-[#CBD5E0]! [&::-webkit-scrollbar-thumb]:rounded [&::-webkit-scrollbar-thumb:hover]:bg-[#A0AEC0]! grow p-[25px] [scrollbar-gutter:stable]">
                    <div v-if="isLoading" class="loading-state text-center p-10 text-[var(--text-color)] text-[1.1em]">
                        <i class="fas fa-spinner fa-spin mr-[10px]"></i> 加载中...
                    </div>
                    <div v-else class="grid gap-5">
                        <!-- ✨ Staggering achieved via inline style -->
                        <a v-for="(category, index) in categories" :key="category.name" href="#" class="border-2 border-transparent rounded-xl p-0 no-underline relative cursor-pointer will-change-transform [transition:transform_0.3s_ease-out,box-shadow_0.3s_ease-out] bg-[var(--card-bg-color)] shadow-[0_4px_15px_var(--shadow-color)] origin-center [&:hover]:[transform:translateY(-4px)_scale(1.05)_rotate(2deg)] [&:hover]:shadow-[10px_10px_25px_var(--shadow-color)] animate-fade-in-up"
                            :style="{ animationDelay: 200 + index * 50 + 'ms' }" @click.prevent="openModal(category)">
                            <div class="bg-transparent rounded-[10px] p-5 text-center text-[var(--text-color)] text-[1.1em] font-medium flex justify-center items-center relative overflow-hidden">
                                <span class="[transition:color_0.3s_ease-out]">{{ category.name }}</span>
                                <span class="bg-[#EDF2F7] text-[#718096] text-[0.8em] py-1 px-[10px] rounded-[20px] ml-[15px] [transition:color_0.3s_ease-out,background-color_0.3s_ease-out]">{{ category.count }}篇</span>
                            </div>
                        </a>
                    </div>
                </div>
            </section>

            <!-- Tags Section -->
            <section class="first-of-type:[animation-delay:0.2s] [&:nth-of-type(2)]:[animation-delay:0.4s] [@media(max-width:900px)]:basis-auto! [@media(max-width:900px)]:h-[50vh] flex-1 animate-fade-in-up">
                <h2 class="[@media(max-width:900px)]:text-[1.5em] [@media(max-width:900px)]:p-5 text-[2em] text-center text-[var(--text-color)] py-[25px] px-5 m-0 shrink-0 [text-shadow:none] border-b border-b-[var(--border-color)]">🏷️ 热门标签</h2>
                <div class="grid-wrapper [@media(max-width:900px)]:p-5 [&::-webkit-scrollbar]:w-2! [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-[#CBD5E0]! [&::-webkit-scrollbar-thumb]:rounded [&::-webkit-scrollbar-thumb:hover]:bg-[#A0AEC0]! grow p-[25px] [scrollbar-gutter:stable]">
                    <div v-if="isLoading" class="loading-state text-center p-10 text-[var(--text-color)] text-[1.1em]">
                        <i class="fas fa-spinner fa-spin mr-[10px]"></i> 加载中...
                    </div>
                    <div v-else class="grid gap-5 grid-cols-[repeat(auto-fill,minmax(130px,1fr))]">
                        <!-- ✨ Staggering achieved via inline style -->
                        <a v-for="(tag, index) in tags" :key="tag.name" href="#" class="border-2 border-transparent rounded-xl p-0 no-underline relative cursor-pointer will-change-transform [transition:transform_0.3s_ease-out,box-shadow_0.3s_ease-out] bg-[var(--card-bg-color)] shadow-[0_4px_15px_var(--shadow-color)] origin-center [&:hover]:[transform:translateY(-4px)_scale(1.05)_rotate(2deg)] [&:hover]:shadow-[10px_10px_25px_var(--shadow-color)] tag-card animate-fade-in-up"
                            :style="{ animationDelay: 800 + index * 50 + 'ms' }" @click.prevent="openModal(tag)">
                            <div class="bg-transparent rounded-[10px] p-5 text-center text-[var(--text-color)] text-[1.1em] font-medium flex justify-center items-center relative overflow-hidden">
                                <span class="[transition:color_0.3s_ease-out]">{{ tag.name }}</span>
                                <span class="absolute top-0 right-[2px] bg-[linear-gradient(135deg,#ff6b6b,#ff8e8e)] text-white rounded-[50%] size-6 flex items-center justify-center text-[0.75rem] font-bold shadow-[0_2px_8px_rgba(255,107,107,0.3)] z-[1]">{{ tag.count }}</span>
                            </div>
                        </a>
                    </div>
                </div>
            </section>
        </main>

        <!-- Modal with Vue Transition -->
        <!-- ✨ Replaced GSAP with Vue's <Transition> component -->
        <Transition enter-active-class="[transition:opacity_0.4s_ease]" leave-active-class="[transition:opacity_0.4s_ease]" enter-from-class="opacity-0" leave-to-class="opacity-0">
            <div v-if="isModalVisible" class="modal-overlay fixed top-0 left-0 w-full h-full bg-[rgba(244,247,252,0.8)] z-[1000] flex justify-center items-center p-5" @click.self="closeModal">
                <Transition enter-active-class="[transition:all_0.4s_cubic-bezier(0.215,0.61,0.355,1)] delay-100" leave-active-class="[transition:all_0.3s_ease-in]" enter-from-class="opacity-0 [transform:translateY(50px)_scale(0.95)]" leave-to-class="opacity-0 [transform:translateY(50px)_scale(0.95)]">
                    <div v-if="isModalVisible" class="[@media(max-width:900px)]:p-5 bg-[var(--card-bg-color)] border border-[var(--border-color)] rounded-2xl py-[25px] px-[30px] w-full max-w-[700px] max-h-[85vh] flex flex-col relative shadow-[0_15px_50px_rgba(45,55,72,0.15)]">
                        <button class="[&:hover]:bg-[#E2E8F0] [&:hover]:[transform:rotate(90deg)] absolute top-[15px] right-[15px] bg-[#EDF2F7] [border:none] text-[var(--text-color)] w-9 h-9 rounded-[50%] text-[24px] leading-[36px] text-center cursor-pointer [transition:background_0.3s,transform_0.3s]" @click="closeModal">×</button>
                        <h3 class="[@media(max-width:900px)]:text-[1.5em] text-[1.8em] text-[var(--text-color)] mx-0 mt-0 mb-5 pb-[15px] pr-10 border-b border-b-[var(--border-color)] [text-shadow:none]">{{ modalTitle }}</h3>
                        <ul class="[&::-webkit-scrollbar]:w-2! [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-[#CBD5E0]! [&::-webkit-scrollbar-thumb]:rounded [&::-webkit-scrollbar-thumb:hover]:bg-[#A0AEC0]! list-none p-0 pr-[15px] m-0 overflow-y-auto grow">
                            <li v-for="(article, index) in modalArticles" :key="index" @click="goToArticle(article)"
                                class="last:border-b-0 [&:hover]:bg-[#F7FAFC] [&:hover]:text-[var(--accent-color-1)] [&:hover]:pl-[25px] before:content-['›'] before:absolute before:left-2 before:top-1/2 before:[transform:translateY(-50%)] before:text-[var(--accent-color-1)] before:opacity-0 before:[transition:opacity_0.3s] before:font-bold [&:hover]:before:opacity-100 border-b border-b-[var(--border-color)] cursor-pointer py-[18px] pr-[10px] pl-5 text-[1.1em] [transition:all_0.3s_ease] relative">
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

/* Preserve the ordered fallback for browsers that do not support overlay. */
.grid-wrapper {
    overflow-x: hidden;
    overflow-y: auto;
    overflow-y: overlay;
}

/* Keep the unprefixed filter; Lightning CSS changes the paired declaration. */
.modal-overlay {
    backdrop-filter: blur(5px);
}
</style>
