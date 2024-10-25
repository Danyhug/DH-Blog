import { userCheck } from '@/api/user';
import { createRouter, createWebHashHistory } from 'vue-router'
// 前端路由组件
const ArticleView = () => import(/* webpackChunkName: "article" */ '../views/frontend/ArticleView.vue');
const MainView = () => import(/* webpackChunkName: "main" */ '../views/frontend/MainView.vue');
const HomeView = () => import(/* webpackChunkName: "home" */ '../views/frontend/HomeView.vue');
const LockView = () => import(/* webpackChunkName: "lock" */ '../views/frontend/LockView.vue');

// 后端路由组件
const AdminView = () => import(/* webpackChunkName: "admin" */ '../views/backend/AdminView.vue');
const PublishView = () => import(/* webpackChunkName: "publish" */ '../views/backend/PublishView.vue');
const ManagerView = () => import(/* webpackChunkName: "manager" */ '../views/backend/ManagerView.vue');
const LoginView = () => import(/* webpackChunkName: "login" */ '../views/backend/LoginView.vue');


const routes = [
  { path: '/', redirect: '/view/home' },
  // 前台页面
  {
    path: '/view', component: HomeView, children:
      [
        {
          path: 'home', component: MainView, name: 'Home', meta: { title: '我的个人纪录' }
        },
        { path: 'article/:id', component: ArticleView, name: 'ArticleInfo' }
      ]
  },
  // 后台页面
  {
    path: '/admin', component: AdminView, name: 'Admin', children:
      [
        // 博客发布
        { path: 'publish', component: PublishView, name: 'publish' },
        // 博客管理
        { path: 'manager', component: ManagerView }
      ]
  },
  // 登录页面
  { path: '/login', component: LoginView, name: 'Login', meta: { title: '登录' } },
  // 加密页面
  { path: '/lock', component: LockView, name: 'Lock', meta: { title: '私密文章' } },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

const isAuthenticated = () => {
  if (localStorage.getItem('token')) {
    return userCheck()
  }
  return router.replace({ name: 'Login' })
}

router.beforeEach(async (to, from, next) => {
  if (to.meta) {
    window.document.title = 'DH-Blog / ' + to.meta.title;
  } else {
    window.document.title = 'DH-Blog的个人纪录';
  }

  if (to.path.startsWith('/admin')) {
    isAuthenticated()
  }

  next()
})

export default router
