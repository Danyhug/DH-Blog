import { createRouter, createWebHashHistory } from 'vue-router'
// 前端路由组件
const ArticleView = () => import(/* webpackChunkName: "article" */ '../views/frontend/ArticleView.vue');
const MainView = () => import(/* webpackChunkName: "main" */ '../views/frontend/MainView.vue');
const HomeView = () => import(/* webpackChunkName: "home" */ '../views/frontend/HomeView.vue');

// 后端路由组件
const AdminView = () => import(/* webpackChunkName: "admin" */ '../views/backend/AdminView.vue');
const PublishView = () => import(/* webpackChunkName: "publish" */ '../views/backend/PublishView.vue');
const ManagerView = () => import(/* webpackChunkName: "manager" */ '../views/backend/ManagerView.vue');


const routes = [
  { path: '/', redirect: '/view/home' },
  // 前台页面
  {
    path: '/view', component: HomeView, children:
      [
        {
          path: 'home', component: MainView, name: 'Home'
        },
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
