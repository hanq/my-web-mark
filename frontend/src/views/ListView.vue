<script setup lang="ts">
/**
 * 列表视图：左侧主题分组 + 右侧书签/账号双 Tab 管理。
 * 支持从浏览器拖拽 URL 到页面直接入库。
 */
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useThemeStore } from '../stores/theme'
import { useDataStore } from '../stores/data'
import { useVaultStore } from '../stores/vault'
import { browserApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import ThemeDialog from '../components/ThemeDialog.vue'
import BookmarkDialog from '../components/BookmarkDialog.vue'
import AccountDialog from '../components/AccountDialog.vue'
import ImportWizard from '../components/ImportWizard.vue'
import type { BookmarkItem, AccountSummary } from '../api'

const themeStore = useThemeStore()
const data = useDataStore()
const vault = useVaultStore()

const tab = ref<'bookmarks' | 'accounts'>('bookmarks')
const showThemeDialog = ref(false)
const editingTheme = ref<null>(null)
const showBookmarkDialog = ref(false)
const editingBookmark = ref<BookmarkItem | null>(null)
const showAccountDialog = ref(false)
const editingAccount = ref<AccountSummary | null>(null)
const showImport = ref(false)
const toast = ref('')
let toastTimer = 0

onMounted(async () => {
  await themeStore.refresh()
  if (themeStore.activeThemeId) {
    await data.load(themeStore.activeThemeId)
  } else if (themeStore.themes.length) {
    themeStore.select(themeStore.themes[0].id)
    await data.load(themeStore.themes[0].id)
  }
})

function flash(msg: string, isErr = false) {
  toast.value = msg
  toastStyle.value = isErr ? 'error' : ''
  window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => (toast.value = ''), 2200)
}
const toastStyle = ref('')

async function pickTheme(id: number) {
  themeStore.select(id)
  await data.load(id)
}

async function removeTheme(id: number, name: string) {
  if (!window.confirm(`删除主题「${name}」及其全部书签与账号？此操作不可恢复。`)) return
  await themeStore.remove(id)
  flash('主题已删除')
  if (themeStore.themes.length) {
    await pickTheme(themeStore.themes[0].id)
  } else {
    await data.load(0)
  }
}

async function openBookmark(bm: BookmarkItem) {
  try {
    await browserApi.openInDefaultBrowser(bm.url)
  } catch (e) {
    flash(String(e instanceof Error ? e.message : e), true)
  }
}

async function removeBookmark(bm: BookmarkItem) {
  if (!window.confirm(`删除书签「${bm.title}」？`)) return
  await data.removeBookmark(bm.id)
  flash('书签已删除')
}

async function removeAccount(acc: AccountSummary) {
  if (!window.confirm(`删除账号「${acc.site} / ${acc.username}」？`)) return
  await data.removeAccount(acc.id)
  flash('账号已删除')
}

async function reveal(acc: AccountSummary) {
  if (!vault.unlocked) {
    flash('请先解锁金库', true)
    return
  }
  await data.revealPassword(acc.id)
}

async function copyPassword(acc: AccountSummary) {
  if (!vault.unlocked) {
    flash('请先解锁金库', true)
    return
  }
  const pwd = data.revealed[acc.id] ?? (await data.revealPassword(acc.id))
  await data.copySecret(pwd)
  flash('密码已复制，15 秒后自动清空剪贴板')
}

async function copyUsername(acc: AccountSummary) {
  await data.copySecret(acc.username)
  flash('用户名已复制')
}

// --- 全局拖拽接收 ---

const dragOver = ref(false)
let dragDepth = 0

function onDragEnter(e: DragEvent) {
  if (!e.dataTransfer?.types.includes('Files') && !e.dataTransfer?.types.includes('text/uri-list') && !e.dataTransfer?.types.includes('text/plain')) return
  dragDepth++
  dragOver.value = true
}
function onDragLeave() {
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragOver.value = false
}
async function onDrop(e: DragEvent) {
  dragDepth = 0
  dragOver.value = false
  const dt = e.dataTransfer
  if (!dt) return
  const text =
    dt.getData('text/uri-list') ||
    dt.getData('text/x-moz-url') ||
    dt.getData('text/html') ||
    dt.getData('text/plain')
  if (!text) return
  e.preventDefault()
  try {
    await data.addDroppedUrl(text)
    flash('书签已收入')
  } catch (err) {
    flash(String(err instanceof Error ? err.message : err), true)
  }
}

const activeName = computed(() => themeStore.activeTheme?.name ?? '')

onUnmounted(() => window.clearTimeout(toastTimer))
</script>

<template>
  <div
    class="list-view"
    @dragenter.prevent="onDragEnter"
    @dragleave="onDragLeave"
    @dragover.prevent
    @drop.prevent="onDrop"
  >
    <!-- 左侧主题栏 -->
    <aside class="side">
      <div class="side-head">
        <span>主题</span>
        <button class="btn ghost icon-btn" title="新建主题" @click="showThemeDialog = true; editingTheme = null">
          <AppIcon name="plus" :size="16" />
        </button>
      </div>
      <div class="side-list">
        <div
          v-for="t in themeStore.themes"
          :key="t.id"
          class="theme-item"
          :class="{ active: t.id === themeStore.activeThemeId }"
          @click="pickTheme(t.id)"
        >
          <span class="theme-dot" :style="{ background: t.color }" />
          <span class="theme-name">{{ t.name }}</span>
          <span class="theme-count">{{ t.bookmarkCount }}</span>
          <button class="btn ghost icon-btn del-btn" title="删除主题" @click.stop="removeTheme(t.id, t.name)">
            <AppIcon name="trash" :size="14" />
          </button>
        </div>
        <div v-if="!themeStore.themes.length" class="empty-mini">暂无主题，点击 + 新建</div>
      </div>
      <button class="btn import-btn" @click="showImport = true">
        <AppIcon name="download" :size="16" />
        从浏览器导入
      </button>
    </aside>

    <!-- 右侧内容 -->
    <section class="content">
      <div class="content-head">
        <h2>{{ activeName || '未选择主题' }}</h2>
        <div class="tabs">
          <button class="tab" :class="{ active: tab === 'bookmarks' }" @click="tab = 'bookmarks'">
            <AppIcon name="bookmark" :size="16" />
            书签 {{ data.bookmarks.length || '' }}
          </button>
          <button class="tab" :class="{ active: tab === 'accounts' }" @click="tab = 'accounts'">
            <AppIcon name="key" :size="16" />
            账号 {{ data.accounts.length || '' }}
          </button>
        </div>
        <div class="head-actions">
          <button
            v-if="tab === 'bookmarks'"
            class="btn primary"
            @click="showBookmarkDialog = true; editingBookmark = null"
          >
            <AppIcon name="plus" :size="16" /> 书签
          </button>
          <button v-else class="btn primary" @click="showAccountDialog = true; editingAccount = null">
            <AppIcon name="plus" :size="16" /> 账号
          </button>
        </div>
      </div>

      <div class="scroll-area">
        <!-- 书签列表 -->
        <div v-if="tab === 'bookmarks'" class="cards">
          <div v-for="bm in data.sortedBookmarks" :key="bm.id" class="card bm-card">
            <span class="favicon" :style="{ background: themeStore.activeTheme?.color }">
              {{ bm.title.slice(0, 1).toUpperCase() }}
            </span>
            <div class="bm-info">
              <div class="bm-title">
                <AppIcon v-if="bm.pinned" name="pin" :size="13" class="pin-ic" />
                {{ bm.title }}
                <span v-if="bm.source !== 'manual'" class="badge muted">{{ bm.source }}</span>
              </div>
              <div class="bm-url" :title="bm.url">{{ bm.url }}</div>
            </div>
            <div class="row-actions">
              <button class="btn ghost icon-btn" title="打开" @click="openBookmark(bm)"><AppIcon name="external" :size="16" /></button>
              <button class="btn ghost icon-btn" title="编辑" @click="editingBookmark = bm; showBookmarkDialog = true"><AppIcon name="edit" :size="16" /></button>
              <button class="btn ghost icon-btn" title="删除" @click="removeBookmark(bm)"><AppIcon name="trash" :size="16" /></button>
            </div>
          </div>
          <div v-if="!data.loading && !data.bookmarks.length" class="empty-state">
            <AppIcon name="bookmark" />
            <span>还没有书签，从浏览器拖一个过来，或点右上角新增</span>
          </div>
        </div>

        <!-- 账号列表 -->
        <div v-else class="cards">
          <div v-for="acc in data.accounts" :key="acc.id" class="card acc-card">
            <div class="acc-main">
              <div class="acc-site">
                {{ acc.site }}
                <a v-if="acc.url" class="acc-link" :title="acc.url" @click="browserApi.openInDefaultBrowser(acc.url)">
                  <AppIcon name="external" :size="12" />
                </a>
              </div>
              <div class="acc-user">
                {{ acc.username }}
                <button class="btn ghost icon-btn mini" title="复制用户名" @click="copyUsername(acc)">
                  <AppIcon name="copy" :size="13" />
                </button>
              </div>
              <div class="acc-note" v-if="acc.note">{{ acc.note }}</div>
            </div>
            <div class="acc-pwd">
              <template v-if="data.revealed[acc.id]">
                <span class="pwd-plain">{{ data.revealed[acc.id] }}</span>
                <button class="btn ghost icon-btn mini" title="隐藏" @click="data.hidePassword(acc.id)">
                  <AppIcon name="eyeOff" :size="14" />
                </button>
              </template>
              <template v-else>
                <span class="pwd-mask">••••••••</span>
                <button class="btn ghost icon-btn mini" :title="vault.unlocked ? '显示' : '金库未解锁'" @click="reveal(acc)">
                  <AppIcon name="eye" :size="14" />
                </button>
              </template>
              <button class="btn ghost icon-btn mini" :title="vault.unlocked ? '复制密码' : '金库未解锁'" @click="copyPassword(acc)">
                <AppIcon name="copy" :size="14" />
              </button>
            </div>
            <div class="row-actions">
              <button class="btn ghost icon-btn" title="编辑" @click="editingAccount = acc; showAccountDialog = true">
                <AppIcon name="edit" :size="16" />
              </button>
              <button class="btn ghost icon-btn" title="删除" @click="removeAccount(acc)">
                <AppIcon name="trash" :size="16" />
              </button>
            </div>
          </div>
          <div v-if="!data.loading && !data.accounts.length" class="empty-state">
            <AppIcon name="key" />
            <span>该主题下暂无账号记录</span>
          </div>
        </div>
      </div>
    </section>

    <!-- 拖拽提示遮罩 -->
    <div v-if="dragOver" class="drop-mask">
      <div class="drop-box">
        <AppIcon name="download" :size="40" />
        <p>松开以收入书签</p>
      </div>
    </div>

    <!-- 对话框 -->
    <ThemeDialog v-if="showThemeDialog" :editing="editingTheme" @close="showThemeDialog = false" />
    <BookmarkDialog
      v-if="showBookmarkDialog"
      :editing="editingBookmark"
      @close="showBookmarkDialog = false"
    />
    <AccountDialog
      v-if="showAccountDialog"
      :editing="editingAccount"
      @close="showAccountDialog = false"
    />
    <ImportWizard v-if="showImport" @close="showImport = false" @done="themeStore.refresh(); data.load(themeStore.activeThemeId)" />

    <div v-if="toast" class="toast" :class="toastStyle">{{ toast }}</div>
  </div>
</template>

<style scoped>
.list-view {
  display: flex;
  height: 100%;
  position: relative;
}
.side {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border);
  background: var(--bg-secondary);
}
.side-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 14px 8px;
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 1px;
}
.icon-btn {
  padding: 4px 6px;
  border: none;
}
.side-list {
  flex: 1;
  overflow: auto;
  padding: 4px 8px;
}
.theme-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.12s;
  position: relative;
}
.theme-item:hover {
  background: var(--bg-card-hover);
}
.theme-item.active {
  background: var(--accent-soft);
}
.theme-dot {
  width: 10px;
  height: 10px;
  border-radius: 3px;
  flex-shrink: 0;
}
.theme-name {
  flex: 1;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.theme-count {
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-input);
  border-radius: 10px;
  padding: 1px 7px;
}
.del-btn {
  opacity: 0;
  transition: opacity 0.12s;
}
.theme-item:hover .del-btn {
  opacity: 1;
}
.empty-mini {
  padding: 20px 10px;
  color: var(--text-muted);
  font-size: 12px;
  text-align: center;
}
.import-btn {
  margin: 10px;
  justify-content: center;
}
.content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.content-head {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
}
.content-head h2 {
  margin: 0;
  font-size: 16px;
  min-width: 80px;
}
.tabs {
  display: flex;
  gap: 4px;
  background: var(--bg-input);
  border-radius: 10px;
  padding: 3px;
}
.tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s;
}
.tab.active {
  background: var(--bg-card);
  color: var(--text-primary);
  box-shadow: var(--shadow-card);
}
.head-actions {
  margin-left: auto;
}
.scroll-area {
  flex: 1;
  overflow: auto;
  padding: 16px 20px;
}
.cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 860px;
}
.bm-card,
.acc-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
}
.favicon {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  font-size: 15px;
  flex-shrink: 0;
}
.bm-info {
  flex: 1;
  min-width: 0;
}
.bm-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
}
.pin-ic {
  color: var(--warning);
}
.bm-url {
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 2px;
}
.row-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
}
.bm-card:hover .row-actions,
.acc-card:hover .row-actions {
  opacity: 1;
}
.acc-main {
  flex: 1;
  min-width: 0;
}
.acc-site {
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.acc-link {
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
}
.acc-link:hover {
  color: var(--accent);
}
.acc-user {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 3px;
  display: flex;
  align-items: center;
  gap: 4px;
}
.mini {
  padding: 2px 4px;
}
.acc-note {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 3px;
}
.acc-pwd {
  display: flex;
  align-items: center;
  gap: 4px;
}
.pwd-mask {
  font-size: 13px;
  letter-spacing: 2px;
  color: var(--text-muted);
}
.pwd-plain {
  font-size: 13px;
  color: var(--text-primary);
  font-family: Consolas, monospace;
}
.drop-mask {
  position: absolute;
  inset: 0;
  background: rgba(10, 14, 26, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  pointer-events: none;
}
.drop-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 36px 60px;
  border: 2px dashed var(--accent);
  border-radius: 18px;
  background: var(--bg-card);
  color: var(--accent);
  box-shadow: var(--shadow-pop);
}
.drop-box p {
  margin: 0;
  color: var(--text-primary);
  font-size: 14px;
}
</style>
