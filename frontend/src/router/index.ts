import { createRouter, createWebHashHistory } from 'vue-router'

/** 上次所在视图（'/'=2.5D 空间、'/list'=书签列表），启动时恢复；设置页不参与记忆 */
const LAST_VIEW_KEY = 'mwm-last-view'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'space', component: () => import('../views/SpaceView.vue') },
    { path: '/list', name: 'list', component: () => import('../views/ListView.vue') },
    { path: '/settings', name: 'settings', component: () => import('../views/SettingsView.vue') },
  ],
})

/**
 * 启动恢复：应用每次启动的初始地址都是默认页 '/'（2.5D 空间），
 * 若上次退出时停在列表页，则把首次导航重定向到 /list。
 * 用守卫而非挂载前 router.replace，因为 app.use(router) 会用初始地址覆盖掉提前发起的导航。
 */
let firstNavigation = true
router.beforeEach((to) => {
  if (firstNavigation) {
    firstNavigation = false
    if (to.name === 'space' && localStorage.getItem(LAST_VIEW_KEY) === '/list') {
      return { name: 'list' }
    }
  }
  return true
})

// 记录空间页/列表页的停留，设置页等临时页面不覆盖记忆
router.afterEach((to) => {
  if (to.path === '/' || to.path === '/list') {
    localStorage.setItem(LAST_VIEW_KEY, to.path)
  }
})

export default router
