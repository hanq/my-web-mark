<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useVaultStore } from './stores/vault'
import { useDataStore } from './stores/data'
import { useSkinStore, type ThemeSkin } from './stores/skin'
import {
  minimiseWindow,
  toggleMaximiseWindow,
  isWindowMaximised,
  quitApp,
} from './api/window_runtime'
import AppIcon from './components/AppIcon.vue'
import VaultLock from './components/VaultLock.vue'

const route = useRoute()
const router = useRouter()
const vault = useVaultStore()
const data = useDataStore()
const skin = useSkinStore()

// 启动时获取金库状态
vault.refresh()

const isList = computed(() => route.name === 'list')
const isSettings = computed(() => route.name === 'settings')

// 窗口最大化状态（标题栏按钮图标切换；resize 时刷新）
const isMax = ref(false)
function refreshMaxState() {
  isWindowMaximised().then((v) => (isMax.value = v))
}
onMounted(() => {
  refreshMaxState()
  window.addEventListener('resize', refreshMaxState)
})
onUnmounted(() => window.removeEventListener('resize', refreshMaxState))

// 金库解锁对话框（顶栏锁定按钮触发）
const showVaultUnlock = ref(false)
// 解锁成功后自动关闭对话框
watch(
  () => vault.unlocked,
  (v) => {
    if (v) showVaultUnlock.value = false
    else data.hideAllPasswords()
  },
)
// 任意页面发起解锁请求（如点击查看密码）时自动弹出解锁框
watch(
  () => vault.unlockRequest,
  (v) => {
    if (v) showVaultUnlock.value = true
  },
)
// 用户关闭解锁框：作废挂起请求，避免发起方一直等待
function closeVaultUnlock() {
  showVaultUnlock.value = false
  vault.settleUnlock(false)
}

const skins: { key: ThemeSkin; label: string }[] = [
  { key: 'dark', label: '暗夜' },
  { key: 'light', label: '简白' },
  { key: 'neon', label: '霓虹' },
  { key: 'chalkboard', label: '黑板粉笔' },
  { key: 'obsidian', label: '黑曜石' },
  { key: 'bamboo', label: '竹子' },
  { key: 'jade', label: '玉石' },
  { key: 'rainbow', label: '彩虹' },
]

function toggleView() {
  router.push(isList.value ? '/' : '/list')
}

/**
 * 标题栏双击空白处 = 最大化/还原（Windows 习惯）。
 * Wails 的拖拽区在鼠标按下时就会启动原生拖动，浏览器的 dblclick 不可靠，
 * 因此自行判定“400ms 内、位移 6px 内的第二次按下”，并在第二次按下时
 * stopPropagation 阻止 Wails 注入的 window mousedown 监听器再次启动拖动。
 * 按钮、皮肤切换、窗口控制等交互区域不参与。
 */
let lastHeadDownAt = 0
let lastHeadDownX = 0
let lastHeadDownY = 0

function onTopBarMouseDown(e: MouseEvent) {
  if (e.button !== 0) return
  const el = e.target as HTMLElement | null
  if (el?.closest('button, a, input, textarea, select, label, .skin-switch, .win-controls')) {
    lastHeadDownAt = 0
    return
  }

  const now = Date.now()
  const isDoubleClick =
    lastHeadDownAt !== 0 &&
    now - lastHeadDownAt <= 400 &&
    Math.abs(e.clientX - lastHeadDownX) <= 6 &&
    Math.abs(e.clientY - lastHeadDownY) <= 6

  if (isDoubleClick) {
    lastHeadDownAt = 0
    e.stopPropagation()
    e.preventDefault()
    toggleMaximiseWindow()
    // 最大化/还原会触发 resize，兜底再刷新一次图标状态
    window.setTimeout(refreshMaxState, 120)
    return
  }

  lastHeadDownAt = now
  lastHeadDownX = e.clientX
  lastHeadDownY = e.clientY
}
</script>

<template>
  <div class="app-shell">
    <header class="top-bar" @mousedown="onTopBarMouseDown">
      <div class="brand">
        <img class="brand-mark" src="/appicon.svg" alt="My Web Mark" draggable="false" />
        <span class="brand-name">My Web Mark</span>
      </div>

      <div class="top-actions">
        <button class="btn ghost" :title="isList ? '切换到 2.5D 空间' : '切换到列表视图'" @click="toggleView">
          <AppIcon :name="isList ? 'cube' : 'list'" />
          <span>{{ isList ? '2.5D 空间' : '列表视图' }}</span>
        </button>

        <div class="skin-switch" title="界面皮肤">
          <AppIcon name="palette" />
          <button
            v-for="s in skins"
            :key="s.key"
            class="skin-dot"
            :class="{ active: skin.skin === s.key }"
            :data-skin="s.key"
            :title="s.label"
            @click="skin.setSkin(s.key)"
          />
        </div>

        <button
          class="btn"
          :class="vault.unlocked ? 'ghost' : 'primary'"
          :title="vault.unlocked ? '锁定金库' : '解锁金库'"
          @click="vault.unlocked ? vault.lock() : (showVaultUnlock = true)"
        >
          <AppIcon :name="vault.unlocked ? 'unlock' : 'lock'" />
          <span>{{ vault.unlocked ? '已解锁' : '已锁定' }}</span>
        </button>

        <button class="btn ghost" title="设置" @click="router.push('/settings')">
          <AppIcon name="settings" />
        </button>
      </div>

      <div class="win-controls">
        <button class="win-btn" title="最小化" @click="minimiseWindow">
          <AppIcon name="minimize" :size="16" />
        </button>
        <button class="win-btn" :title="isMax ? '还原' : '最大化'" @click="toggleMaximiseWindow">
          <AppIcon :name="isMax ? 'restore' : 'maximize'" :size="15" />
        </button>
        <button class="win-btn close" title="关闭" @click="quitApp">
          <AppIcon name="close" :size="16" />
        </button>
      </div>
    </header>

    <main class="main-area">
      <router-view />
    </main>

    <!-- 未设置管理员密码时，首次引导设置 -->
    <VaultLock v-if="vault.setupRequired" mode="setup" />
    <!-- 锁定后主动解锁（顶栏按钮触发） -->
    <VaultLock v-else-if="showVaultUnlock && !vault.unlocked" mode="unlock" @close="closeVaultUnlock" />
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0 0 18px;
  height: 46px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  /* Wails 无边框窗口拖拽区：标题栏整体可拖动，交互元素单独标记 no-drag */
  --wails-draggable: drag;
}
.top-actions button,
.top-actions .skin-switch {
  --wails-draggable: no-drag;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 100%;
}
.brand-mark {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  display: block;
  user-select: none;
}
.brand-name {
  font-weight: 600;
  font-size: 15px;
  letter-spacing: 0.5px;
}
.top-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 10px;
}
.win-controls {
  display: flex;
  height: 100%;
  --wails-draggable: no-drag;
}
.win-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 100%;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}
.win-btn:hover {
  background: var(--bg-card-hover);
  color: var(--text-primary);
}
.win-btn.close:hover {
  background: #e81123;
  color: #fff;
}
.skin-switch {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg-card);
  color: var(--text-secondary);
}
.skin-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
  transition: transform 0.15s;
}
.skin-dot:hover {
  transform: scale(1.2);
}
.skin-dot.active {
  border-color: var(--text-primary);
}
.skin-dot[data-skin='dark'] {
  background: #2b3550;
}
.skin-dot[data-skin='light'] {
  background: #e8ecf4;
}
.skin-dot[data-skin='neon'] {
  background: linear-gradient(135deg, #b26bff, #00e5b0);
}
.skin-dot[data-skin='chalkboard'] {
  background: #2a3d34;
}
.skin-dot[data-skin='obsidian'] {
  background: linear-gradient(135deg, #131320, #7c6cff);
}
.skin-dot[data-skin='bamboo'] {
  background: #7cb65e;
}
.skin-dot[data-skin='jade'] {
  background: linear-gradient(135deg, #0f1e1c, #38b894);
}
.skin-dot[data-skin='rainbow'] {
  background: linear-gradient(135deg, #ff6b9d, #ffb648, #4dd4ac, #7c6cff);
}
.main-area {
  flex: 1;
  min-height: 0;
  position: relative;
}
</style>
