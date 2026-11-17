<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useVaultStore } from './stores/vault'
import { useSkinStore, type ThemeSkin } from './stores/skin'
import AppIcon from './components/AppIcon.vue'
import VaultLock from './components/VaultLock.vue'

const route = useRoute()
const router = useRouter()
const vault = useVaultStore()
const skin = useSkinStore()

// 启动时获取金库状态
vault.refresh()

const isList = computed(() => route.name === 'list')
const isSettings = computed(() => route.name === 'settings')

const skins: { key: ThemeSkin; label: string }[] = [
  { key: 'dark', label: '暗夜' },
  { key: 'light', label: '简白' },
  { key: 'neon', label: '霓虹' },
]

function toggleView() {
  router.push(isList.value ? '/' : '/list')
}
</script>

<template>
  <div class="app-shell">
    <header class="top-bar">
      <div class="brand">
        <span class="brand-mark"><AppIcon name="cube" :size="22" /></span>
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
          @click="vault.unlocked ? vault.lock() : router.push('/settings')"
        >
          <AppIcon :name="vault.unlocked ? 'unlock' : 'lock'" />
          <span>{{ vault.unlocked ? '已解锁' : '已锁定' }}</span>
        </button>

        <button class="btn ghost" title="设置" @click="router.push('/settings')">
          <AppIcon name="settings" />
        </button>
      </div>
    </header>

    <main class="main-area">
      <router-view />
    </main>

    <!-- 未设置管理员密码时，首次引导设置 -->
    <VaultLock v-if="vault.setupRequired" mode="setup" />
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
  padding: 10px 18px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  -webkit-app-region: drag;
}
.top-bar button,
.top-bar .skin-switch {
  -webkit-app-region: no-drag;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
}
.brand-mark {
  display: flex;
  color: var(--accent);
}
.brand-name {
  font-weight: 600;
  font-size: 15px;
  letter-spacing: 0.5px;
}
.top-actions {
  display: flex;
  align-items: center;
  gap: 10px;
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
.main-area {
  flex: 1;
  min-height: 0;
  position: relative;
}
</style>
