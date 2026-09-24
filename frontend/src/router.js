import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'

const Login = () => import('./views/Login.vue')
const AdminLayout = () => import('./views/admin/Layout.vue')
const UserLayout = () => import('./views/user/Layout.vue')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    {
      path: '/admin',
      component: AdminLayout,
      meta: { auth: true, role: 'admin' },
      children: [
        { path: '', redirect: '/admin/dashboard' },
        { path: 'dashboard', component: () => import('./views/admin/Dashboard.vue'), meta: { perm: 'dashboard.view' } },
        { path: 'orders', component: () => import('./views/admin/Orders.vue'), meta: { perm: 'order.view' } },
        { path: 'users', component: () => import('./views/admin/Users.vue'), meta: { perm: ['user.view', 'admin.manage'] } },
        { path: 'wallets', component: () => import('./views/admin/Wallets.vue'), meta: { perm: 'wallet.view' } },
        { path: 'withdrawals', component: () => import('./views/admin/Withdrawals.vue'), meta: { perm: 'withdraw.view' } },
        { path: 'reconcile', component: () => import('./views/admin/Reconcile.vue'), meta: { perm: 'reconcile.view' } },
        { path: 'audit', component: () => import('./views/admin/Audit.vue'), meta: { perm: 'audit.view' } },
        { path: 'roles', component: () => import('./views/admin/Roles.vue'), meta: { perm: 'role.manage' } },
      ],
    },
    {
      path: '/app',
      component: UserLayout,
      meta: { auth: true, role: 'user' },
      children: [
        { path: '', redirect: '/app/orders' },
        { path: 'orders', component: () => import('./views/user/Orders.vue') },
        { path: 'wallet', component: () => import('./views/user/Wallet.vue') },
      ],
    },
    { path: '/', redirect: '/login' },
  ],
})

function firstAdminPath(auth) {
  const candidates = [
    ['dashboard.view', '/admin/dashboard'],
    ['order.view', '/admin/orders'],
    ['user.view', '/admin/users'],
    ['admin.manage', '/admin/users'],
    ['wallet.view', '/admin/wallets'],
    ['withdraw.view', '/admin/withdrawals'],
    ['reconcile.view', '/admin/reconcile'],
    ['audit.view', '/admin/audit'],
    ['role.manage', '/admin/roles'],
  ]
  for (const [perm, path] of candidates) {
    if (auth.hasPerm(perm)) return path
  }
  return '/login'
}

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.auth && !auth.isLoggedIn) return '/login'
  if (to.meta.role === 'admin' && !auth.isAdmin) return '/app/orders'
  if (to.meta.role === 'user' && auth.isAdmin) return firstAdminPath(auth)

  if (to.meta.role === 'admin' && auth.isAdmin && (!auth.user?.permissions || to.path === '/admin' || to.path === '/admin/')) {
    try {
      await auth.fetchMe()
    } catch {
      auth.logout()
      return '/login'
    }
  }

  const perm = to.meta.perm
  if (perm && auth.isAdmin) {
    const codes = Array.isArray(perm) ? perm : [perm]
    if (!auth.hasPerm(...codes)) return firstAdminPath(auth)
  }
  return true
})

export default router
