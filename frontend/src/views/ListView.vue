<script setup lang="ts">
/**
 * 列表视图：左侧主题分组 + 右侧书签/账号双 Tab 管理。
 * 支持从浏览器拖拽 URL 到页面直接入库。
 */
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useThemeStore } from '../stores/theme'
import { useDataStore } from '../stores/data'
import { useVaultStore } from '../stores/vault'
import { browserApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import ThemeDialog from '../components/ThemeDialog.vue'
import BookmarkDialog from '../components/BookmarkDialog.vue'
import AccountDialog from '../components/AccountDialog.vue'
import ImportWizard from '../components/ImportWizard.vue'
import TagManageDialog from '../components/TagManageDialog.vue'
import ThemeExportDialog from '../components/ThemeExportDialog.vue'
import ThemeIcon from '../components/ThemeIcon.vue'
import { exportApi, type ThemeSummary } from '../api'
import type { BookmarkItem, AccountSummary } from '../api'

const themeStore = useThemeStore()
const data = useDataStore()
const vault = useVaultStore()

const tab = ref<'bookmarks' | 'accounts'>('bookmarks')
const showThemeDialog = ref(false)
const editingTheme = ref<ThemeSummary | null>(null)
const showBookmarkDialog = ref(false)
const editingBookmark = ref<BookmarkItem | null>(null)
const showAccountDialog = ref(false)
const editingAccount = ref<AccountSummary | null>(null)
const showImport = ref(false)
const showTagManage = ref(false)
const showExport = ref(false)
const exportTheme = ref<ThemeSummary | null>(null)
/** 主题项操作菜单（更多按钮/右键呼出，Teleport 到 body） */
const menuTheme = ref<ThemeSummary | null>(null)
const menuPos = ref({ x: 0, y: 0 })

function openItemMenu(e: MouseEvent, t: ThemeSummary) {
  // 预留菜单尺寸做视口边界夹紧，避免右侧/底部溢出
  const MENU_W = 140
  const MENU_H = 130
  const x = Math.min(e.clientX, window.innerWidth - MENU_W - 8)
  const y = Math.min(e.clientY, window.innerHeight - MENU_H - 8)
  menuPos.value = { x: Math.max(8, x), y: Math.max(8, y) }
  menuTheme.value = t
}

function menuAction(action: 'edit' | 'export' | 'delete') {
  const t = menuTheme.value
  menuTheme.value = null
  if (!t) return
  if (action === 'edit') {
    editingTheme.value = t
    showThemeDialog.value = true
  } else if (action === 'export') {
    exportTheme.value = t
    showExport.value = true
  } else {
    removeTheme(t.id, t.name)
  }
}

/** 侧栏收起态（收起只显示图标，hover 提示名称） */
const sideCollapsed = ref(localStorage.getItem('mwm-side-collapsed') === '1')
function toggleSide() {
  sideCollapsed.value = !sideCollapsed.value
  localStorage.setItem('mwm-side-collapsed', sideCollapsed.value ? '1' : '0')
}
const importingFile = ref(false)
const toast = ref('')
let toastTimer = 0

onMounted(async () => {
  await themeStore.refresh()
  // 恢复上次选中的主题（已删除则回退第一个）
  const savedTheme = Number(localStorage.getItem('mwm-last-theme')) || 0
  let target = themeStore.themes.some((t) => t.id === savedTheme) ? savedTheme : 0
  if (!target && themeStore.themes.length) target = themeStore.themes[0].id
  if (target) {
    themeStore.select(target)
    await data.load(target)
    // 恢复上次选中的标签（须属于当前主题）
    const savedTag = Number(localStorage.getItem('mwm-last-tag')) || 0
    if (savedTag && data.tags.some((t) => t.id === savedTag)) data.setTagFilter(savedTag)
  }
  // 恢复上次的书签/账号 Tab
  const savedTab = localStorage.getItem('mwm-last-tab')
  if (savedTab === 'accounts') tab.value = 'accounts'
  // 后台补齐书签站点图标
  data.bookmarks.forEach((b) => data.ensureFavicon(b.url))
})

// Tab 切换持久化（书签/账号）
watch(tab, (t) => localStorage.setItem('mwm-last-tab', t))

// 书签列表变化时为新条目补齐图标
watch(
  () => data.bookmarks.length,
  () => data.bookmarks.forEach((b) => data.ensureFavicon(b.url)),
)

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

/** 从文件导入主题书签（HTML/JSON 自动检测） */
async function importDataFile() {
  importingFile.value = true
  try {
    const res = await exportApi.importTheme()
    if (!res) return // 用户取消
    flash(`已导入「${res.themeName}」：${res.imported} 条书签、${res.tagCount} 个标签${res.skipped ? `，跳过 ${res.skipped} 条重复` : ''}`)
    await themeStore.refresh()
    await data.loadAllBookmarks()
    await data.loadAllTags()
  } catch (e) {
    flash(String(e instanceof Error ? e.message : e), true)
  } finally {
    importingFile.value = false
  }
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
    const tip = await browserApi.openInDefaultBrowser(bm.url)
    if (tip) flash(tip) // 如"未找到 Chrome，已使用系统默认浏览器打开"
  } catch (e) {
    flash(String(e instanceof Error ? e.message : e), true)
  }
}

/** 显示用 URL：去掉协议头与尾部斜杠 */
function prettyUrl(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

// --- 卡片拖拽排序 ---

const dragId = ref<number | null>(null)
const dropTargetId = ref<number | null>(null)
const dropPos = ref<'before' | 'after' | null>(null)

function clearDragState() {
  dragId.value = null
  dropTargetId.value = null
  dropPos.value = null
}

function onCardDragStart(e: DragEvent, bm: BookmarkItem) {
  dragId.value = bm.id
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    // 自定义类型：全局拖拽入库逻辑据此排除书签排序拖拽
    e.dataTransfer.setData('application/x-mwm-bookmark', String(bm.id))
  }
}

function onCardDragOver(e: DragEvent, bm: BookmarkItem) {
  const dragBm = data.bookmarks.find((b) => b.id === dragId.value)
  if (!dragBm || dragBm.pinned !== bm.pinned || dragId.value === bm.id) return
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  dropTargetId.value = bm.id
  dropPos.value = e.clientX < rect.left + rect.width / 2 ? 'before' : 'after'
}

function onCardDragLeave() {
  // 仅当完全离开卡片时清除指示（dragleave 会在子元素间频繁触发，
  // 简化处理：留到 dragover 其他卡片或 dragend 时统一更新）
}

async function onCardDrop(e: DragEvent, target: BookmarkItem) {
  if (dragId.value == null || dropTargetId.value !== target.id || !dropPos.value) return
  e.preventDefault()
  const drag = dragId.value
  const pos = dropPos.value
  clearDragState()
  await data.reorderBookmarks(drag, target.id, pos)
}

function onCardDragEnd() {
  clearDragState()
}

/** 主题拖拽排序状态 */
const themeDragId = ref<number | null>(null)
const themeDropTarget = ref<number | null>(null)
const themeDropPos = ref<'before' | 'after' | null>(null)

function onThemeDragStart(e: DragEvent, t: ThemeSummary) {
  themeDragId.value = t.id
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('application/x-mwm-theme', String(t.id))
  }
}

function onThemeDragOver(e: DragEvent, t: ThemeSummary) {
  if (themeDragId.value == null || themeDragId.value === t.id) return
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  themeDropTarget.value = t.id
  themeDropPos.value = e.clientY < rect.top + rect.height / 2 ? 'before' : 'after'
}

async function onThemeDrop(e: DragEvent, target: ThemeSummary) {
  if (themeDragId.value == null || themeDropTarget.value !== target.id || !themeDropPos.value) return
  e.preventDefault()
  const drag = themeDragId.value
  const pos = themeDropPos.value
  themeDragId.value = null
  themeDropTarget.value = null
  themeDropPos.value = null
  await themeStore.reorderThemes(drag, target.id, pos)
}

function onThemeDragEnd() {
  themeDragId.value = null
  themeDropTarget.value = null
  themeDropPos.value = null
}

/** 数字键 0-9 快捷打开对应编号书签（编号显示在卡片左上角） */
function onGlobalKeydown(e: KeyboardEvent) {
  if (tab.value !== 'bookmarks') return
  const t = e.target as HTMLElement | null
  if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT' || t.isContentEditable)) return
  if (e.ctrlKey || e.altKey || e.metaKey || e.shiftKey) return
  const m = /^(?:Digit|Numpad)([0-9])$/.exec(e.code)
  if (!m) return
  const bm = data.filteredBookmarks[Number(m[1])]
  if (!bm) return
  e.preventDefault()
  openBookmark(bm)
}
onMounted(() => window.addEventListener('keydown', onGlobalKeydown))
onUnmounted(() => window.removeEventListener('keydown', onGlobalKeydown))

async function removeBookmark(bm: BookmarkItem) {
  if (!window.confirm(`删除书签「${bm.title}」？`)) return
  await data.removeBookmark(bm.id)
  flash('书签已删除')
}

async function removeAccount(acc: AccountSummary) {
  if (!window.confirm(`删除「${accLabel(acc)}」？`)) return
  await data.removeAccount(acc.id)
  flash('账号已删除')
}

async function reveal(acc: AccountSummary) {
  // 锁定时就地弹出解锁框，解锁后继续展示，无需退出重进
  if (!(await vault.requestUnlock('查看密码需要先解锁金库'))) return
  try {
    await data.revealPassword(acc.id)
  } catch (e) {
    flash(String(e instanceof Error ? e.message : e), true)
  }
}

async function copyPassword(acc: AccountSummary) {
  if (!(await vault.requestUnlock('复制密码需要先解锁金库'))) return
  try {
    // 直接读明文但不清空到展示缓存：复制不应把卡片上的小眼睛打开
    const pwd = await data.fetchPassword(acc.id)
    await data.copySecret(pwd)
    flash('密码已复制，15 秒后自动清空剪贴板')
  } catch (e) {
    flash(String(e instanceof Error ? e.message : e), true)
  }
}

/** 一键复制概要信息（名称/用户名/密码/网址/备注/标签），有密码时自动请求解锁。 */
async function copyAccountAll(acc: AccountSummary) {
  let pwd = ''
  if (acc.hasPwd) {
    if (!(await vault.requestUnlock('复制密码需要先解锁金库'))) return
    try {
      pwd = await data.fetchPassword(acc.id)
    } catch (e) {
      flash(String(e instanceof Error ? e.message : e), true)
      return
    }
  }
  const lines = [
    `名称：${accLabel(acc)}`,
    acc.username ? `用户名：${acc.username}` : '',
    pwd ? `密码：${pwd}` : '',
    acc.url ? `网址：${acc.url}` : '',
    acc.note ? `备注：${acc.note}` : '',
    acc.tags.length ? `标签：${acc.tags.map((t) => t.name).join('、')}` : '',
  ].filter(Boolean)
  await data.copySecret(lines.join('\n'))
  flash(pwd ? '名称、用户名、密码等已复制，15 秒后清空剪贴板' : '概要信息已复制，15 秒后清空剪贴板')
}

async function copyUsername(acc: AccountSummary) {
  await data.copySecret(acc.username)
  flash('用户名已复制')
}

// --- 账号列表：类型 / 搜索 / 标签筛选 ---

const ACC_KIND_LABEL: Record<string, string> = {
  web: '网站',
  device: '设备',
  wifi: 'WiFi',
  other: '其他',
}
const ACC_KIND_ICON: Record<string, string> = {
  web: 'globe',
  device: 'monitor',
  wifi: 'wifi',
  other: 'key',
}

/** 账号列表搜索关键词 */
const accQuery = ref('')
/** 账号列表当前筛选标签；null 为全部 */
const accTagId = ref<number | null>(null)

/** 账号展示名：站点名 → 用户名 → 网址域名 → 备注首行 → 类型名 */
function accLabel(acc: AccountSummary): string {
  if (acc.site) return acc.site
  if (acc.username) return acc.username
  if (acc.url) return prettyUrl(acc.url)
  if (acc.note) return acc.note.split('\n')[0].slice(0, 24)
  return `${ACC_KIND_LABEL[acc.kind] ?? '账号'}凭据`
}

const filteredAccounts = computed(() => {
  const q = accQuery.value.trim().toLowerCase()
  return data.accounts.filter((a) => {
    if (accTagId.value != null && !a.tags.some((t) => t.id === accTagId.value)) return false
    if (!q) return true
    const hay = [a.site, a.username, a.url, a.note, ...a.tags.map((t) => t.name)].join(' ').toLowerCase()
    return hay.includes(q)
  })
})

/** 打开账号关联的书签（优先按书签打开，已删除则直接用网址） */
function openAccountTarget(acc: AccountSummary) {
  if (acc.bookmarkId) {
    const bm = data.bookmarks.find((b) => b.id === acc.bookmarkId)
    if (bm) {
      openBookmark(bm)
      return
    }
  }
  if (acc.url) browserApi.openInDefaultBrowser(acc.url)
}

// --- 全局拖拽接收 ---

const dragOver = ref(false)
let dragDepth = 0

function onDragEnter(e: DragEvent) {
  // 卡片/主题排序拖拽（自定义类型）不触发入库遮罩
  const types = e.dataTransfer?.types
  if (types?.includes('application/x-mwm-bookmark') || types?.includes('application/x-mwm-theme')) return
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
  // 卡片/主题排序拖拽：由各自的 drop 处理，不做 URL 入库
  if (dt.types.includes('application/x-mwm-bookmark') || dt.types.includes('application/x-mwm-theme')) return
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
/** 筛选后的书签数（Tab 徽标显示用） */
const filteredCount = computed(() =>
  data.activeTagId == null ? data.bookmarks.length : data.filteredBookmarks.length,
)

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
    <aside class="side" :class="{ collapsed: sideCollapsed }">
      <div class="side-head">
        <span v-if="!sideCollapsed" class="side-title">主题</span>
        <button
          class="btn ghost icon-btn"
          :title="sideCollapsed ? '展开侧栏' : '收起侧栏'"
          @click="toggleSide"
        >
          <AppIcon :name="sideCollapsed ? 'chevronRight' : 'chevronLeft'" :size="16" />
        </button>
      </div>
      <div class="side-list">
        <div
          v-for="t in themeStore.themes"
          :key="t.id"
          class="theme-item"
          :class="{
            active: t.id === themeStore.activeThemeId,
            dragging: themeDragId === t.id,
            'drop-before': themeDropTarget === t.id && themeDropPos === 'before',
            'drop-after': themeDropTarget === t.id && themeDropPos === 'after',
          }"
          :data-name="t.name"
          :title="sideCollapsed ? t.name : undefined"
          draggable="true"
          @click="pickTheme(t.id)"
          @contextmenu.prevent="openItemMenu($event, t)"
          @dragstart="onThemeDragStart($event, t)"
          @dragover="onThemeDragOver($event, t)"
          @drop="onThemeDrop($event, t)"
          @dragend="onThemeDragEnd"
        >
          <ThemeIcon v-if="t.icon" :name="t.icon" :size="17" class="theme-ic" :style="{ color: t.color }" />
          <span v-else class="theme-dot" :style="{ background: t.color }" />
          <template v-if="!sideCollapsed">
            <span class="theme-name">{{ t.name }}</span>
            <span class="theme-count">{{ t.bookmarkCount }}</span>
            <button class="btn ghost icon-btn action-btn" title="更多操作" @click.stop="openItemMenu($event, t)">
              <AppIcon name="more" :size="14" />
            </button>
          </template>
        </div>
        <div v-if="!themeStore.themes.length" class="empty-mini">暂无主题</div>
      </div>
      <div class="side-foot">
        <button class="foot-btn" title="新建主题" @click="showThemeDialog = true; editingTheme = null">
          <AppIcon name="plus" :size="16" />
          <span v-if="!sideCollapsed">新建主题</span>
        </button>
        <button class="foot-btn" title="从浏览器导入" @click="showImport = true">
          <AppIcon name="download" :size="16" />
          <span v-if="!sideCollapsed">从浏览器导入</span>
        </button>
        <button class="foot-btn" title="导入书签文件" :disabled="importingFile" @click="importDataFile">
          <AppIcon name="folder" :size="16" />
          <span v-if="!sideCollapsed">{{ importingFile ? '导入中…' : '导入书签文件' }}</span>
        </button>
      </div>
    </aside>

    <!-- 右侧内容 -->
    <section class="content">
      <div class="content-head">
        <h2>{{ activeName || '未选择主题' }}</h2>
        <div class="tabs">
          <button class="tab" :class="{ active: tab === 'bookmarks' }" @click="tab = 'bookmarks'">
            <AppIcon name="bookmark" :size="16" />
            书签 {{ filteredCount }}
          </button>
          <button class="tab" :class="{ active: tab === 'accounts' }" @click="tab = 'accounts'">
            <AppIcon name="key" :size="16" />
            账号密码 {{ data.accounts.length || '' }}
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
            <AppIcon name="plus" :size="16" /> 条目
          </button>
        </div>
      </div>

      <!-- 标签筛选条（书签列表上方） -->
      <div v-if="tab === 'bookmarks'" class="tag-bar">
        <button class="chip" :class="{ active: data.activeTagId == null }" @click="data.setTagFilter(null)">
          全部
        </button>
        <button
          v-for="t in data.tags"
          :key="t.id"
          class="chip"
          :class="{ active: data.activeTagId === t.id }"
          @click="data.setTagFilter(t.id)"
        >
          <span class="chip-dot" :style="{ background: t.color || 'var(--accent)' }" />
          {{ t.name }}
        </button>
        <button class="chip manage" title="管理标签" @click="showTagManage = true">
          <AppIcon name="settings" :size="13" /> 标签
        </button>
      </div>

      <!-- 账号 Tab：搜索 + 标签筛选（与书签筛选互不影响） -->
      <div v-else class="tag-bar acc-bar">
        <div class="acc-search">
          <AppIcon name="search" :size="14" />
          <input v-model="accQuery" class="acc-search-input" placeholder="搜索名称、用户名、网址、备注、标签" />
          <button v-if="accQuery" class="btn ghost icon-btn mini" title="清空" @click="accQuery = ''">
            <AppIcon name="close" :size="13" />
          </button>
        </div>
        <button class="chip" :class="{ active: accTagId == null }" @click="accTagId = null">全部</button>
        <button
          v-for="t in data.tags"
          :key="t.id"
          class="chip"
          :class="{ active: accTagId === t.id }"
          @click="accTagId = accTagId === t.id ? null : t.id"
        >
          <span class="chip-dot" :style="{ background: t.color || 'var(--accent)' }" />
          {{ t.name }}
        </button>
      </div>

      <div class="scroll-area">
        <!-- 书签列表：自适应多列卡片网格，支持拖拽排序 -->
        <template v-if="tab === 'bookmarks'">
          <div v-if="data.filteredBookmarks.length" class="bm-grid">
            <div
              v-for="(bm, idx) in data.filteredBookmarks"
              :key="bm.id"
              class="card bm-card"
              :class="{
                dragging: dragId === bm.id,
                'drop-before': dropTargetId === bm.id && dropPos === 'before',
                'drop-after': dropTargetId === bm.id && dropPos === 'after',
              }"
              draggable="true"
              title="点击打开，拖动可排序"
              @click="openBookmark(bm)"
              @dragstart="onCardDragStart($event, bm)"
              @dragover="onCardDragOver($event, bm)"
              @dragleave="onCardDragLeave"
              @drop="onCardDrop($event, bm)"
              @dragend="onCardDragEnd"
            >
              <div class="bm-head">
                <span v-if="idx < 10" class="bm-num" :title="`按数字键 ${idx} 快速打开`">{{ idx }}</span>
                <span
                  class="favicon"
                  :style="{ background: data.favicons[bm.url] ? '#ffffff' : themeStore.activeTheme?.color }"
                >
                  <img
                    v-if="data.favicons[bm.url]"
                    :src="data.favicons[bm.url]"
                    alt=""
                    @error="data.faviconFailed(bm.url)"
                  />
                  <span v-else>{{ bm.title.slice(0, 1).toUpperCase() }}</span>
                </span>
                <div class="bm-title" :title="bm.title">{{ bm.title }}</div>
                <AppIcon v-if="bm.pinned" name="pin" :size="14" class="pin-ic" />
              </div>
              <div class="bm-url" :title="bm.url">{{ prettyUrl(bm.url) }}</div>
              <div class="bm-foot">
                <div v-if="bm.tags.length" class="bm-tags">
                  <span
                    v-for="t in bm.tags.slice(0, 2)"
                    :key="t.id"
                    class="mini-tag"
                    :title="`标签：${t.name}`"
                  >
                    <span class="mini-tag-dot" :style="{ background: t.color || 'var(--accent)' }" />
                    {{ t.name }}
                  </span>
                  <span v-if="bm.tags.length > 2" class="mini-tag more">+{{ bm.tags.length - 2 }}</span>
                </div>
                <div v-else class="bm-tags" />
                <div class="row-actions" @click.stop>
                  <button class="btn ghost icon-btn" title="编辑" @click="editingBookmark = bm; showBookmarkDialog = true">
                    <AppIcon name="edit" :size="15" />
                  </button>
                  <button class="btn ghost icon-btn" title="删除" @click="removeBookmark(bm)">
                    <AppIcon name="trash" :size="15" />
                  </button>
                </div>
              </div>
            </div>
          </div>
          <div v-if="!data.loading && !data.bookmarks.length" class="empty-state">
            <AppIcon name="bookmark" />
            <span>还没有书签，从浏览器拖一个过来，或点右上角新增</span>
          </div>
          <div v-else-if="!data.loading && data.bookmarks.length && !data.filteredBookmarks.length" class="empty-state">
            <AppIcon name="bookmark" />
            <span>当前标签下暂无书签</span>
          </div>
          <p v-else-if="data.filteredBookmarks.length" class="drag-hint">拖动卡片可调整顺序 · 点击卡片打开</p>
        </template>

        <!-- 账号列表 -->
        <div v-else class="cards">
          <div v-for="acc in filteredAccounts" :key="acc.id" class="card acc-card">
            <div class="acc-main">
              <div class="acc-site">
                <span class="acc-kind" :title="ACC_KIND_LABEL[acc.kind] ?? '其他'">
                  <AppIcon :name="ACC_KIND_ICON[acc.kind] ?? 'key'" :size="13" />
                </span>
                <span class="acc-name" :title="accLabel(acc)">{{ accLabel(acc) }}</span>
                <a
                  v-if="acc.url"
                  class="acc-link"
                  :title="acc.bookmarkId ? '打开关联书签' : '在浏览器中打开'"
                  @click="openAccountTarget(acc)"
                >
                  <AppIcon :name="acc.bookmarkId ? 'bookmark' : 'external'" :size="12" />
                </a>
              </div>
              <div v-if="acc.note || acc.tags.length" class="acc-sub">
                <span v-if="acc.note" class="acc-note" :title="acc.note">{{ acc.note }}</span>
                <div v-if="acc.tags.length" class="acc-tags">
                  <span
                    v-for="t in acc.tags"
                    :key="t.id"
                    class="mini-tag"
                    :title="`标签：${t.name}`"
                    :style="{ borderColor: t.color, color: t.color }"
                  >
                    <span class="mini-tag-dot" :style="{ background: t.color || 'var(--accent)' }" />
                    {{ t.name }}
                  </span>
                </div>
              </div>
            </div>

            <!-- 凭据区：用户名 / 密码 分栏对齐，互不挤压 -->
            <div class="acc-creds">
              <div class="acc-field">
                <span class="acc-field-label">用户名</span>
                <div class="acc-field-value">
                  <span class="acc-text" :title="acc.username">{{ acc.username || '—' }}</span>
                  <button
                    v-if="acc.username"
                    class="btn ghost icon-btn mini"
                    title="复制用户名"
                    @click="copyUsername(acc)"
                  >
                    <AppIcon name="copy" :size="13" />
                  </button>
                </div>
              </div>
              <div class="acc-field">
                <span class="acc-field-label">密码</span>
                <div class="acc-field-value">
                  <template v-if="!acc.hasPwd">
                    <span class="pwd-empty">未保存</span>
                  </template>
                  <template v-else-if="data.revealed[acc.id]">
                    <span class="pwd-plain" :title="data.revealed[acc.id]">{{ data.revealed[acc.id] }}</span>
                    <button class="btn ghost icon-btn mini" title="隐藏" @click="data.hidePassword(acc.id)">
                      <AppIcon name="eyeOff" :size="14" />
                    </button>
                  </template>
                  <template v-else>
                    <span class="pwd-mask">••••••••</span>
                    <button
                      class="btn ghost icon-btn mini"
                      :title="vault.unlocked ? '显示' : '点击解锁后显示'"
                      @click="reveal(acc)"
                    >
                      <AppIcon name="eye" :size="14" />
                    </button>
                  </template>
                  <button
                    v-if="acc.hasPwd"
                    class="btn ghost icon-btn mini"
                    :title="vault.unlocked ? '复制密码' : '点击解锁后复制'"
                    @click="copyPassword(acc)"
                  >
                    <AppIcon name="copy" :size="14" />
                  </button>
                </div>
              </div>
            </div>

            <div class="row-actions">
              <button
                class="btn ghost icon-btn"
                title="复制概要（名称、用户名、密码、网址、备注）"
                @click="copyAccountAll(acc)"
              >
                <AppIcon name="clipboard" :size="16" />
              </button>
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
            <span>该主题下暂无账号条目。可添加网站账号、电脑/设备密码、WiFi 密码等，无需解锁即可浏览。</span>
          </div>
          <div
            v-else-if="!data.loading && data.accounts.length && !filteredAccounts.length"
            class="empty-state"
          >
            <AppIcon name="search" />
            <span>没有符合条件的条目，可清空搜索或切换标签</span>
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
    <ThemeExportDialog
      v-if="showExport && exportTheme"
      :theme-id="exportTheme.id"
      :theme-name="exportTheme.name"
      @close="showExport = false"
    />
    <TagManageDialog v-if="showTagManage" @close="showTagManage = false" />

    <div v-if="toast" class="toast" :class="toastStyle">{{ toast }}</div>

    <!-- 主题项操作菜单（挂 body 避免侧栏 overflow 裁剪） -->
    <Teleport to="body">
      <div v-if="menuTheme" class="menu-mask" @click="menuTheme = null" @contextmenu.prevent="menuTheme = null" />
      <div
        v-if="menuTheme"
        class="item-menu"
        :style="{ left: menuPos.x + 'px', top: menuPos.y + 'px' }"
      >
        <button class="menu-item" @click="menuAction('edit')">
          <AppIcon name="edit" :size="14" /> 编辑
        </button>
        <button class="menu-item" @click="menuAction('export')">
          <AppIcon name="download" :size="14" /> 导出
        </button>
        <div class="menu-sep" />
        <button class="menu-item danger" @click="menuAction('delete')">
          <AppIcon name="trash" :size="14" /> 删除
        </button>
      </div>
    </Teleport>
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
  transition: width 0.18s ease;
}
.side.collapsed {
  width: 60px;
}
.theme-item {
  position: relative;
}
.theme-item.dragging {
  opacity: 0.45;
}
.theme-item.drop-before {
  box-shadow: 0 -2px 0 0 var(--accent);
}
.theme-item.drop-after {
  box-shadow: 0 2px 0 0 var(--accent);
}
.theme-ic {
  flex-shrink: 0;
}
.side.collapsed .theme-item {
  flex-direction: column;
  gap: 4px;
  padding: 9px 0;
  justify-content: center;
}
.side.collapsed .theme-item:hover::after {
  content: attr(data-name);
  position: absolute;
  left: calc(100% + 12px);
  top: 50%;
  transform: translateY(-50%);
  background: var(--bg-card);
  border: 1px solid var(--border-strong);
  color: var(--text-primary);
  font-size: 12px;
  padding: 5px 12px;
  border-radius: 8px;
  white-space: nowrap;
  box-shadow: var(--shadow-pop);
  z-index: 60;
  pointer-events: none;
}
.side-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 10px 8px 14px;
}
.side.collapsed .side-head {
  justify-content: center;
  padding: 10px 6px 8px;
}
.side-title {
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
.action-btn {
  opacity: 0;
  transition: opacity 0.12s;
}
.theme-item:hover .action-btn {
  opacity: 1;
}
/* 主题项操作菜单（Teleport 挂 body，fixed 定位） */
.menu-mask {
  position: fixed;
  inset: 0;
  z-index: 150;
}
.item-menu {
  position: fixed;
  min-width: 132px;
  background: var(--bg-card);
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  box-shadow: var(--shadow-pop);
  padding: 5px;
  z-index: 151;
}
.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 10px;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  text-align: left;
  transition: background 0.12s;
}
.menu-item:hover {
  background: var(--bg-card-hover);
}
.menu-item.danger {
  color: var(--danger);
}
.menu-sep {
  height: 1px;
  background: var(--border);
  margin: 4px 6px;
}
/* 侧栏底部：统一按钮容器（展开/收起共用规格） */
.side-foot {
  border-top: 1px solid var(--border);
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.foot-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 13px;
  text-align: left;
  transition: all 0.12s;
}
.foot-btn:hover {
  background: var(--bg-card-hover);
  color: var(--text-primary);
}
.foot-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.side.collapsed .side-foot {
  padding: 8px 6px;
}
.side.collapsed .foot-btn {
  justify-content: center;
  padding: 8px 0;
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
  padding: 14px 20px 10px;
}
.tag-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 0 20px 12px;
  border-bottom: 1px solid var(--border);
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--bg-card);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.12s;
}
.chip:hover {
  border-color: var(--accent);
  color: var(--text-primary);
}
.chip.active {
  background: var(--accent-soft);
  border-color: var(--accent);
  color: var(--text-primary);
}
.chip.manage {
  border-style: dashed;
  color: var(--text-muted);
}
.chip-dot {
  width: 9px;
  height: 9px;
  border-radius: 3px;
}
/* 账号 Tab 工具条：搜索框 + 标签筛选 */
.acc-bar {
  row-gap: 8px;
}
.acc-search {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--bg-input);
  color: var(--text-muted);
  min-width: 220px;
  flex: 0 1 300px;
}
.acc-search:focus-within {
  border-color: var(--accent);
  color: var(--text-secondary);
}
.acc-search-input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 12px;
  padding: 2px 0;
}
.favicon img {
  width: 22px;
  height: 22px;
  object-fit: contain;
  border-radius: 4px;
}
.bm-tags {
  display: flex;
  gap: 6px;
  margin-top: 4px;
  flex-wrap: wrap;
}
.mini-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text-muted);
  background: var(--bg-input);
  border-radius: 8px;
  padding: 1px 7px;
}
.mini-tag-dot {
  width: 6px;
  height: 6px;
  border-radius: 2px;
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

/* ---------- 书签多列网格（拖拽排序） ---------- */
.bm-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(248px, 1fr));
  gap: 12px;
}
.bm-card {
  display: flex;
  flex-direction: column;
  gap: 7px;
  padding: 12px;
  cursor: pointer;
  transition: transform 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}
.bm-card:hover {
  transform: translateY(-2px);
  border-color: var(--accent);
  box-shadow: var(--shadow-pop);
}
.bm-card.dragging {
  opacity: 0.45;
  transform: scale(0.98);
}
.bm-card.drop-before {
  box-shadow: -3px 0 0 0 var(--accent);
}
.bm-card.drop-after {
  box-shadow: 3px 0 0 0 var(--accent);
}
.bm-head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.favicon {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  font-size: 14px;
  flex-shrink: 0;
}
.bm-title {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pin-ic {
  color: var(--warning);
  flex-shrink: 0;
}
.bm-url {
  font-size: 11px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  unicode-bidi: plaintext;
  text-align: left;
}
.bm-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 24px;
}
.bm-tags {
  display: flex;
  gap: 5px;
  min-width: 0;
  overflow: hidden;
}
.mini-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text-muted);
  background: var(--bg-input);
  border-radius: 8px;
  padding: 1px 7px;
  white-space: nowrap;
}
.mini-tag.more {
  padding: 1px 6px;
}
.mini-tag-dot {
  width: 6px;
  height: 6px;
  border-radius: 2px;
}
.row-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
  flex-shrink: 0;
}
.bm-card:hover .row-actions,
.acc-card:hover .row-actions {
  opacity: 1;
}
/* 编号：内嵌键帽风格（kbd 语义），提示快捷键而非消息数 */
.bm-num {
  flex-shrink: 0;
  min-width: 18px;
  padding: 3px 4px;
  border-radius: 5px;
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-muted);
  font-size: 10px;
  font-weight: 600;
  font-family: Consolas, monospace;
  line-height: 1;
  text-align: center;
  align-self: center;
  pointer-events: none;
}
.drag-hint {
  margin: 12px 2px 0;
  font-size: 11px;
  color: var(--text-muted);
}
.acc-main {
  flex: 1 1 auto;
  min-width: 0;
}
/* 账号卡片：主体 | 凭据分栏 | 操作，三段式横向布局，避免全部挤在左侧 */
.acc-card {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 12px 14px;
}
.acc-site {
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.acc-sub {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 4px;
}
.acc-kind {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  border-radius: 6px;
  background: var(--accent-soft);
  color: var(--accent);
}
.acc-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.acc-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}
.acc-tags .mini-tag {
  border: 1px solid currentColor;
  border-radius: 999px;
  padding: 1px 7px;
}
.acc-link {
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
}
.acc-link:hover {
  color: var(--accent);
}
/* 凭据分栏：用户名 / 密码各自带标签，列间留固定间隔 */
.acc-creds {
  display: flex;
  align-items: flex-start;
  gap: 22px;
  flex: 0 0 auto;
  padding-left: 4px;
}
.acc-field {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
.acc-field-label {
  font-size: 10px;
  letter-spacing: 0.4px;
  color: var(--text-muted);
}
.acc-field-value {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 22px;
  font-size: 12px;
  color: var(--text-secondary);
}
.acc-field-value .acc-text {
  max-width: 170px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mini {
  padding: 2px 4px;
}
.acc-note {
  font-size: 11px;
  color: var(--text-muted);
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pwd-mask {
  font-size: 13px;
  letter-spacing: 2px;
  color: var(--text-muted);
}
.pwd-empty {
  font-size: 11px;
  color: var(--text-muted);
  opacity: 0.8;
}
.pwd-plain {
  font-size: 13px;
  color: var(--text-primary);
  font-family: Consolas, monospace;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 窄窗口：凭据区换行到下一行并保持间隔 */
@media (max-width: 900px) {
  .acc-card {
    flex-wrap: wrap;
    row-gap: 10px;
  }
  .acc-creds {
    order: 3;
    width: 100%;
    padding-left: 0;
  }
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
