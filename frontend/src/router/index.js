import { createRouter, createWebHistory } from 'vue-router'
import { useUser } from '../stores/user'

const routes = [
  { path: '/', name: 'square', component: () => import('../views/SquareView.vue'), meta: { public: true } },
  { path: '/install', name: 'install', component: () => import('../views/InstallView.vue'), meta: { public: true } },
  { path: '/login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
  { path: '/register', name: 'register', component: () => import('../views/RegisterView.vue'), meta: { public: true } },
  {
    path: '/dashboard',
    component: () => import('../views/Layout.vue'),
    children: [
      { path: '', name: 'dashboard', component: () => import('../views/DashboardView.vue') },
      { path: 'keys', name: 'keys', component: () => import('../views/KeysView.vue') },
      { path: 'logs', name: 'logs', component: () => import('../views/LogsView.vue') },
      { path: 'profile', name: 'profile', component: () => import('../views/ProfileView.vue') },
    ],
  },
  {
    path: '/admin',
    component: () => import('../views/admin/AdminLayout.vue'),
    meta: { admin: true },
    children: [
      { path: '', name: 'admin-dash', component: () => import('../views/admin/AdminDash.vue') },
      { path: 'config', name: 'admin-config', component: () => import('../views/admin/AdminConfig.vue') },
      { path: 'users', name: 'admin-users', component: () => import('../views/admin/AdminUsers.vue') },
      { path: 'groups', name: 'admin-groups', component: () => import('../views/admin/AdminGroups.vue') },
      { path: 'models', name: 'admin-models', component: () => import('../views/admin/AdminModels.vue') },
      { path: 'templates', name: 'admin-templates', component: () => import('../views/admin/AdminTemplates.vue') },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to) => {
  const user = useUser()
  if (to.meta.admin && !user.isAdmin) return { name: 'login', query: { redirect: to.fullPath } }
  if (!to.meta.public && !user.loggedIn) return { name: 'login', query: { redirect: to.fullPath } }
  return true
})

export default router
