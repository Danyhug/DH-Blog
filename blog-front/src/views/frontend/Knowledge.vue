<template>
    <div>
        <div id="particles-js"></div>
        <main class="main-container">
            <section class="content-section">
                <h2 class="section-title">📂 文章分类</h2>
                <div class="grid-wrapper">
                    <div v-if="isLoading" class="loading-state">
                        <i class="fas fa-spinner fa-spin"></i> 加载中...
                    </div>
                    <div v-else class="grid-container">
                        <!-- ✨ Staggering achieved via inline style -->
                        <a v-for="(category, index) in categories" :key="category.name" href="#" class="card"
                            :style="{ animationDelay: 800 + index * 50 + 'ms' }" @click.prevent="openModal(category)">
                            <div class="card-content">
                                <span>{{ category.name }}</span>
                                <span class="count">{{ category.count }}</span>
                            </div>
                        </a>
                    </div>
                </div>
            </section>

            <!-- Tags Section -->
            <section class="content-section tag-section" style="flex: 1">
                <h2 class="section-title">🏷️ 热门标签</h2>
                <div class="grid-wrapper">
                    <div v-if="isLoading" class="loading-state">
                        <i class="fas fa-spinner fa-spin"></i> 加载中...
                    </div>
                    <div v-else class="grid-container">
                        <!-- ✨ Staggering achieved via inline style -->
                        <a v-for="(tag, index) in tags" :key="tag.name" href="#" class="card"
                            :style="{ animationDelay: 800 + index * 50 + 'ms' }" @click.prevent="openModal(tag)">
                            <div class="card-content">
                                <span>{{ tag.name }}</span>
                            </div>
                        </a>
                    </div>
                </div>
            </section>
        </main>

        <!-- Modal with Vue Transition -->
        <!-- ✨ Replaced GSAP with Vue's <Transition> component -->
        <Transition name="modal-fade">
            <div v-if="isModalVisible" class="modal-overlay" @click.self="closeModal">
                <Transition name="modal-zoom">
                    <div v-if="isModalVisible" class="modal-content">
                        <button class="modal-close-btn" @click="closeModal">×</button>
                        <h3 class="modal-title">{{ modalTitle }}</h3>
                        <ul class="article-list">
                            <li v-for="(article, index) in modalArticles" :key="index">
                                <div>{{ article.title }}</div>
                                <div class="article-info">
                                    <span><i class="fas fa-eye"></i> {{ article.views }} 阅读</span>
                                    <span><i class="fas fa-file-word"></i> {{ article.wordNum }} 字</span>
                                    <span><i class="fas fa-calendar-alt"></i> {{ article.createTime }}</span>
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

// --- Reactive State Management ---
const allData = ref([]);
const isLoading = ref(false);
const isModalVisible = ref(false);
const modalTitle = ref('');
const modalArticles = ref([]);

const categories = computed(() => allData.value.filter(item => item.type === 'category'));
const tags = computed(() => allData.value.filter(item => item.type === 'tag'));

// --- API Methods ---
const loadTaxonomies = async () => {
    isLoading.value = true;
    try {
        const data = await getAllTaxonomies();
        allData.value = data.map(item => ({
            ...item,
            count: 0 // 默认计数为0，后续可以添加统计
        }));
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
            title: article.title,
            views: article.views,
            wordNum: article.wordNum,
            createTime: article.createTime
        }));
    } catch (error) {
        console.error('获取文章列表失败:', error);
        modalArticles.value = [{ title: '加载文章失败，请稍后重试' }];
    }
    
    isModalVisible.value = true;
};

const closeModal = () => {
    isModalVisible.value = false;
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

<style>
.loading-state {
    text-align: center;
    padding: 40px;
    color: var(--text-color);
    font-size: 1.1em;
}

.loading-state i {
    margin-right: 10px;
}

.article-info {
    font-size: 0.85em;
    color: #666;
    margin-top: 6px;
}

.article-info span {
    margin-right: 12px;
}

.article-info i {
    margin-right: 4px;
}

/* ✨ New CSS Animations to replace GSAP */
@keyframes fadeInUp {
    from {
        opacity: 0;
        transform: translateY(50px);
    }

    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.content-section {
    animation: fadeInUp 1s cubic-bezier(0.215, 0.61, 0.355, 1) both;
}

.content-section:nth-of-type(1) {
    animation-delay: 0.2s;
}

.content-section:nth-of-type(2) {
    animation-delay: 0.4s;
}

.card {
    /* Link card to the animation, but it will be delayed by inline style */
    animation: fadeInUp 1s cubic-bezier(0.215, 0.61, 0.355, 1) both;
}

/* Modal Fade Transition (for the overlay) */
.modal-fade-enter-active,
.modal-fade-leave-active {
    transition: opacity 0.4s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
    opacity: 0;
}

/* Modal Zoom Transition (for the content) */
.modal-zoom-enter-active {
    transition: all 0.4s cubic-bezier(0.215, 0.61, 0.355, 1);
    /* ease-out-expo */
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


/* --- All other styles remain the same --- */
:root {
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

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    margin: 0;
    background-color: var(--bg-color);
    color: var(--text-color);
    height: 100vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

#particles-js {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    z-index: 0;
}

.main-container {
    display: flex;
    flex-grow: 1;
    padding: 40px;
    gap: 40px;
    z-index: 1;
    position: relative;
    height: calc(100vh - 80px);
}

.section-categories {
    flex-basis: 40%;
}

.section-tags {
    flex-basis: 60%;
}

.section-title {
    font-size: 2em;
    text-align: center;
    color: var(--text-color);
    padding: 25px 20px;
    margin: 0;
    flex-shrink: 0;
    text-shadow: none;
    border-bottom: 1px solid var(--border-color);
}

.grid-wrapper {
    overflow-y: auto;
    overflow-x: hidden;
    padding: 25px;
    flex-grow: 1;
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

.grid-container {
    display: grid;
    gap: 20px;
}

.section-categories .grid-container {
    grid-template-columns: minmax(0, 450px);
    justify-content: center;
}

.section-tags .grid-container {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    /* 在小屏幕上保持网格布局 */
}

/* 横向排布 */
@media (min-width: 901px) {
    .section-tags .grid-container {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
    }
    
    .section-tags .grid-container .card {
        width: auto;
        white-space: nowrap;
    }
}

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

.section-categories .card-content {
    justify-content: space-between;
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

.card-content>span:first-child {
    transition: color 0.3s ease-out;
}

.card:hover {
    transform: translateY(-4px) scale(1.05) rotate(2deg);
    box-shadow: 10px 10px 25px var(--shadow-color);
    border-radius: 12px;
}

.section-tags .card:hover {
    border-color: var(--accent-color-1);
}

.section-categories .card:hover {
    border-image-source: linear-gradient(120deg,
            var(--accent-color-1),
            var(--accent-color-2));
    border-image-slice: 1;
}

.section-categories .card:hover .card-content>span:first-child {
    color: var(--accent-color-1);
}

.section-categories .card:hover .count {
    background-color: hsl(180, 75%, 95%);
    color: var(--accent-color-1);
}

.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(244, 247, 252, 0.8);
    backdrop-filter: blur(5px);
    -webkit-backdrop-filter: blur(5px);
    z-index: 1000;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 20px;
}

.modal-content {
    background: var(--card-bg-color);
    border: 1px solid var(--border-color);
    border-radius: 16px;
    padding: 25px 30px;
    width: 100%;
    max-width: 700px;
    max-height: 85vh;
    display: flex;
    flex-direction: column;
    position: relative;
    box-shadow: 0 15px 50px rgba(45, 55, 72, 0.15);
}

.modal-close-btn {
    position: absolute;
    top: 15px;
    right: 15px;
    background: #EDF2F7;
    border: none;
    color: var(--text-color);
    width: 36px;
    height: 36px;
    border-radius: 50%;
    font-size: 24px;
    line-height: 36px;
    text-align: center;
    cursor: pointer;
    transition: background 0.3s, transform 0.3s;
}

.modal-close-btn:hover {
    background: #E2E8F0;
    transform: rotate(90deg);
}

.modal-title {
    font-size: 1.8em;
    color: var(--text-color);
    margin: 0 0 20px 0;
    padding-bottom: 15px;
    padding-right: 40px;
    border-bottom: 1px solid var(--border-color);
    text-shadow: none;
}

.article-list {
    list-style: none;
    padding: 0;
    margin: 0;
    overflow-y: auto;
    flex-grow: 1;
    padding-right: 15px;
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

.article-list li {
    padding: 18px 10px 18px 20px;
    border-bottom: 1px solid var(--border-color);
    font-size: 1.1em;
    cursor: pointer;
    transition: all 0.3s ease;
    position: relative;
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

@media (max-width: 900px) {
    body {
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

    .section-categories .grid-container {
        grid-template-columns: 1fr;
    }

    .modal-content {
        padding: 20px;
    }

    .modal-title {
        font-size: 1.5em;
    }
}

.tag-section .grid-container {
    grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
}
</style>