import { createRouter, createWebHashHistory } from 'vue-router'
import ArticleView from '../views/frontend/ArticleView.vue'
import MainView from '../views/frontend/MainView.vue'
import HomeView from '../views/frontend/HomeView.vue'

import AdminView from '../views/backend/AdminView.vue'
import PublishView from '../views/backend/PublishView.vue'
import ManagerView from '../views/backend/ManagerView.vue'

const routes = [
  { path: '/', redirect: '/view/home' },
  // 前台页面
  {
    path: '/view', component: HomeView, children:
      [
        { path: 'home', component: MainView },
        { path: 'article/:id', component: ArticleView }
      ]
  },
  // 后台页面
  {
    path: '/admin', component: AdminView, children:
      [
        // 博客发布
        { path: 'publish', component: PublishView, name: 'publish' },
        // 博客管理
        { path: 'manager', component: ManagerView }
      ]
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
