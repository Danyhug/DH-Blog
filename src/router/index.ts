import { createRouter, createWebHistory} from 'vue-router'
import ArticleView from '../views/ArticleView.vue'
import MainView from '../views/MainView.vue'

const routes = [
  { path: '/', component: MainView },
  { path: '/article/:id', component: ArticleView },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
