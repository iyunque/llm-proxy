import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue')
  },
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue')
      },
      {
        path: 'providers',
        name: 'Providers',
        component: () => import('../views/Providers.vue')
      },
      {
        path: 'endpoints',
        name: 'Endpoints',
        component: () => import('../views/Endpoints.vue')
      },
      {
        path: 'stats',
        name: 'Stats',
        component: () => import('../views/Stats.vue')
      },
      {
        path: 'test',
        name: 'APITest',
        component: () => import('../views/APITest.vue')
      },
      {
        path: 'user-center',
        name: 'UserCenter',
        component: () => import('../views/UserCenter.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.name !== 'Login' && !token) {
    next({ name: 'Login' })
  } else {
    next()
  }
})

export default router
