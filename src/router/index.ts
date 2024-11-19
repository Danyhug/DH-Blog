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
const DashBoardView = () => import(/* webpackChunkName: "dashboard" */ '../views/backend/DashBoardView.vue');
const CommentView = () => import(/* webpackChunkName: "comment" */ '../views/backend/CommentView.vue');


import NProgress from 'nprogress'
import 'nprogress/nprogress.css'

// 顶部进度条配置
NProgress.configure({
  easing: 'ease', // 动画方式
  speed: 600, // 递增进度条的速度
  showSpinner: false, // 是否显示加载ico
  trickleSpeed: 200, // 自动递增间隔
  parent: 'body' //指定进度条的父容器
})

const routes = [
  { path: '/', redirect: '/view/home' },
  // 前台页面
  {
    path: '/view', component: HomeView, children:
      [
        { path: 'home', component: MainView, name: 'Home', meta: { title: '我的个人纪录' } },
        { path: 'article/:id', component: ArticleView, name: 'ArticleInfo', meta: { title: '文章详情' } }
      ]
  },
  // 后台页面
  {
    path: '/admin', redirect: '/admin/dashboard', component: AdminView, name: 'Admin', meta: { title: '后台管理' }, children:
      [
        // 仪表盘
        { path: 'dashboard', component: DashBoardView, name: 'Dashboard', meta: { title: '后台 仪表盘' } },
        // 博客发布
        { path: 'publish', component: PublishView, name: 'publish', meta: { title: '后台 博客发布' } },
        // 博客管理
        { path: 'manager', component: ManagerView, name: 'manager', meta: { title: '后台 博客管理' } },
        // 评论管理
        { path: 'comment', component: CommentView, name: 'comment', meta: { title: '后台 评论管理' } },
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

router.beforeEach(async (to, _, next) => {
  NProgress.start()
  if (to.meta.title) {
    window.document.title = 'DH-Blog / ' + to.meta.title;
  } else {
    window.document.title = 'DH-Blog的个人纪录';
  }

  if (to.path.startsWith('/admin')) {
    isAuthenticated()
  }

  next()
})

router.afterEach(() => {
  NProgress.done()
})

export default router
