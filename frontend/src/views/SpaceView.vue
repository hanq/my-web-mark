<script setup lang="ts">
/**
 * 3D 卡片画廊视图（Cover Flow）。
 * 数据走响应式 store（allBookmarks/allTags），主题 chips 切换画廊页，
 * 底部标签筛选条 + 中央卡片信息条（打开/编辑/删除）。
 */
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { SpaceEngine } from '../scenes/engine'
import { useThemeStore } from '../stores/theme'
import { useDataStore } from '../stores/data'
import { useSkinStore } from '../stores/skin'
import { browserApi, pageTitleApi, type BookmarkItem } from '../api'
import AppIcon from '../components/AppIcon.vue'
import BookmarkDialog from '../components/BookmarkDialog.vue'
import ImportWizard from '../components/ImportWizard.vue'

const themeStore = useThemeStore()
const data = useDataStore()
const skin = useSkinStore()

const containerEl = ref<HTMLDivElement | null>(null)
const engine = ref<SpaceEngine | null>(null)
const showBookmarkDialog = ref(false)
const editingBookmark = ref<BookmarkItem | null>(null)
/** 拖拽入库预填的 URL（对话框打开时带入） */
const newBookmarkUrl = ref('')
const newBookmarkTitle = ref('')

/** 拖 URL 到场景：解析后打开新书签对话框（预填 URL） */
async function onDropUrl(e: DragEvent) {
  const dt = e.dataTransfer
  if (!dt) return
  // 卡片/主题排序拖拽不处理
  if (dt.types.includes('application/x-mwm-bookmark') || dt.types.includes('application/x-mwm-theme')) return
  const text = dt.getData('text/uri-list') || dt.getData('text/html') || dt.getData('text/plain')
  if (!text) return
  e.preventDefault()
  try {
    const parsed = await browserApi.parseDroppedURL(text)
    let title = parsed.title
    try {
      const fetched = await pageTitleApi.fetch(parsed.rawUrl)
      if (fetched) title = fetched
    } catch {
      /* 抓取失败用 host */
    }
    newBookmarkUrl.value = parsed.rawUrl
    newBookmarkTitle.value = title
    showBookmarkDialog.value = true
  } catch (err) {
    flash(String(err instanceof Error ? err.message : err), true)
  }
}
const showImport = ref(false)
const toast = ref('')
const toastType = ref('')
let toastTimer = 0

/** 当前画廊主题（默认第一个有书签的主题） */
const currentThemeId = ref(0)
/** 画廊中央卡片（引擎回调） */
const centerCard = ref<BookmarkItem | null>(null)

function flash(msg: string, isErr = false) {
  toast.value = msg
  toastType.value = isErr ? 'error' : ''
  window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => (toast.value = ''), 2200)
}

/** 当前主题 + 标签筛选后的画廊卡片（顺序沿用置顶优先排序） */
const sceneCards = computed(() => {
  return data.allBookmarks
    .filter(
      (b) =>
        (currentThemeId.value === 0 || b.themeId === currentThemeId.value) &&
        (data.activeTagId == null || b.tags.some((t) => t.id === data.activeTagId)),
    )
    .sort((a, b) => Number(b.pinned) - Number(a.pinned))
})

const activeTheme = computed(() =>
  themeStore.themes.find((t) => t.id === currentThemeId.value) ?? null,
)

/** 响应式空态判定（修复：此前依赖非响应式变量导致空态卡死） */
const hasSceneData = computed(() => data.allBookmarks.length > 0)
const currentEmpty = computed(() => hasSceneData.value && sceneCards.value.length === 0)

/** 推送当前卡片集与主题色到引擎 */
function pushCards() {
  const th = activeTheme.value
  engine.value?.setCards(sceneCards.value, th?.color ?? '#4f8cff')
  // 后台补齐站点图标并回写卡片纹理
  sceneCards.value.forEach((bm) => {
    data.ensureFavicon(bm.url).then((src) => {
      if (src) engine.value?.setCardFavicon(bm.id, src)
    })
  })
}

/** 全量刷新：主题、标签、全部书签 */
async function refreshAll() {
  await themeStore.refresh()
  await data.loadAllTags()
  await data.loadAllBookmarks()
  if (!currentThemeId.value && themeStore.themes.length) {
    // 恢复上次选中的主题（已删除则回退到第一个有书签的主题）
    const saved = Number(localStorage.getItem('mwm-last-theme')) || 0
    const savedValid = themeStore.themes.some((t) => t.id === saved)
    const first = savedValid
      ? themeStore.themes.find((t) => t.id === saved)!
      : themeStore.themes.find((t) => t.bookmarkCount > 0) ?? themeStore.themes[0]
    currentThemeId.value = first.id
    themeStore.select(first.id)
  }
  // 恢复上次选中的标签筛选（须存在于全局标签中）
  if (data.activeTagId == null) {
    const savedTag = Number(localStorage.getItem('mwm-last-tag')) || 0
    if (savedTag && data.allTags.some((t) => t.id === savedTag)) data.setTagFilter(savedTag)
  }
  pushCards()
}

onMounted(async () => {
  if (!containerEl.value) return
  engine.value = new SpaceEngine(containerEl.value, {
    onBookmarkClick: openBookmark,
    onCenterChange: (bm) => (centerCard.value = bm),
  })
  skin.registerSceneSync((s) => engine.value?.applySkin(s))
  await refreshAll()
})

onUnmounted(() => {
  engine.value?.dispose()
  engine.value = null
  window.clearTimeout(toastTimer)
})

// 数据/主题/筛选变化 → 重推卡片
watch(sceneCards, pushCards)
watch(() => themeStore.themes.map((t) => `${t.id}:${t.color}`).join(','), () => {
  if (currentThemeId.value && !themeStore.themes.some((t) => t.id === currentThemeId.value)) {
    currentThemeId.value = themeStore.themes[0]?.id ?? 0
  }
  pushCards()
})

function pickTheme(id: number) {
  currentThemeId.value = id
  themeStore.select(id) // 同步全局选中并持久化
}

async function openBookmark(bm: BookmarkItem) {
  try {
    await browserApi.openInDefaultBrowser(bm.url)
  } catch (e) {
    flash(String(e instanceof Error ? e.message : e), true)
  }
}

async function removeCenter() {
  const bm = centerCard.value
  if (!bm) return
  if (!window.confirm(`删除书签「${bm.title}」？`)) return
  await data.removeBookmark(bm.id)
  await data.loadAllBookmarks()
  flash('书签已删除')
}
</script>

<template>
  <div class="space-view">
    <div ref="containerEl" class="scene-container" @dragover.prevent @drop.prevent="onDropUrl" />

    <!-- 顶部主题切换条 -->
    <div class="theme-bar" v-if="themeStore.themes.length">
      <button
        v-for="t in themeStore.themes"
        :key="t.id"
        class="chip"
        :class="{ active: t.id === currentThemeId }"
        @click="pickTheme(t.id)"
      >
        <span class="chip-dot" :style="{ background: t.color }" />
        {{ t.name }}
        <span class="chip-count">{{ t.bookmarkCount }}</span>
      </button>
    </div>

    <!-- 中央卡片信息条 -->
    <div v-if="centerCard" class="center-info">
      <div class="ci-main">
        <div class="ci-title">
          {{ centerCard.title }}
          <AppIcon v-if="centerCard.pinned" name="pin" :size="13" class="ci-pin" />
        </div>
        <div class="ci-url">{{ centerCard.url }}</div>
        <div v-if="centerCard.tags.length" class="ci-tags">
          <span v-for="t in centerCard.tags" :key="t.id" class="ci-tag">
            <span class="ci-tag-dot" :style="{ background: t.color || 'var(--accent)' }" />
            {{ t.name }}
          </span>
        </div>
      </div>
      <div class="ci-actions">
        <button class="btn primary" @click="openBookmark(centerCard)">
          <AppIcon name="external" :size="15" /> 打开
        </button>
        <button class="btn" @click="editingBookmark = centerCard; showBookmarkDialog = true">
          <AppIcon name="edit" :size="15" />
        </button>
        <button class="btn danger" @click="removeCenter">
          <AppIcon name="trash" :size="15" />
        </button>
      </div>
    </div>

    <!-- 底部标签筛选条 -->
    <div class="scene-tag-bar" v-if="data.allTags.length">
      <button class="chip" :class="{ active: data.activeTagId == null }" @click="data.setTagFilter(null)">
        全部
      </button>
      <button
        v-for="t in data.allTags"
        :key="t.id"
        class="chip"
        :class="{ active: data.activeTagId === t.id }"
        @click="data.setTagFilter(t.id)"
      >
        <span class="chip-dot" :style="{ background: t.color || 'var(--accent)' }" />
        {{ t.name }}
      </button>
    </div>

    <!-- 操作提示 -->
    <div class="scene-hint">滚轮 / ←→ 切换 · 拖拽滑动 · 点击中央卡打开 · Enter 打开</div>
    <button class="btn corner-btn" title="回到第一张" @click="engine?.resetView()">
      <AppIcon name="refresh" :size="15" />
    </button>

    <!-- 空态 -->
    <div v-if="!hasSceneData" class="scene-empty">
      <AppIcon name="cube" :size="46" />
      <h3>你的书签空间还是空的</h3>
      <p>新建主题并添加书签，或从浏览器一键导入，它们会以 3D 卡片画廊的方式呈现在这里。</p>
      <button class="btn primary" @click="showImport = true">
        <AppIcon name="download" :size="15" /> 从浏览器导入
      </button>
    </div>
    <div v-else-if="currentEmpty" class="scene-empty soft">
      <AppIcon name="bookmark" :size="40" />
      <h3>该主题暂无书签</h3>
      <p>选择其他主题，或为新主题添加书签。</p>
      <button class="btn primary" @click="showBookmarkDialog = true">
        <AppIcon name="plus" :size="15" /> 新书签
      </button>
    </div>

    <BookmarkDialog
      v-if="showBookmarkDialog"
      :editing="editingBookmark"
      :preset-url="newBookmarkUrl"
      :preset-title="newBookmarkTitle"
      @close="showBookmarkDialog = false; editingBookmark = null; newBookmarkUrl = ''; newBookmarkTitle = ''"
    />
    <ImportWizard v-if="showImport" @close="showImport = false" @done="refreshAll()" />

    <div v-if="toast" class="toast" :class="toastType">{{ toast }}</div>
  </div>
</template>

<style scoped>
.space-view {
  position: relative;
  height: 100%;
  overflow: hidden;
}
.scene-container {
  position: absolute;
  inset: 0;
}
.theme-bar {
  position: absolute;
  top: 14px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 8px;
  max-width: calc(100% - 40px);
  overflow-x: auto;
  padding: 6px 10px;
  background: color-mix(in srgb, var(--bg-card) 82%, transparent);
  backdrop-filter: blur(6px);
  border: 1px solid var(--border);
  border-radius: 12px;
  z-index: 10;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
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
.chip-dot {
  width: 9px;
  height: 9px;
  border-radius: 3px;
}
.chip-count {
  font-size: 10px;
  color: var(--text-muted);
  background: var(--bg-input);
  border-radius: 8px;
  padding: 0 6px;
}
.center-info {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  bottom: 64px;
  display: flex;
  align-items: center;
  gap: 16px;
  max-width: min(640px, calc(100% - 60px));
  padding: 10px 16px;
  background: color-mix(in srgb, var(--bg-card) 88%, transparent);
  backdrop-filter: blur(6px);
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  box-shadow: var(--shadow-pop);
  z-index: 10;
}
.ci-main {
  min-width: 0;
}
.ci-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ci-pin {
  color: var(--warning);
}
.ci-url {
  font-size: 11px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 2px;
}
.ci-tags {
  display: flex;
  gap: 5px;
  margin-top: 4px;
  flex-wrap: wrap;
}
.ci-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text-muted);
  background: var(--bg-input);
  border-radius: 8px;
  padding: 1px 7px;
}
.ci-tag-dot {
  width: 6px;
  height: 6px;
  border-radius: 2px;
}
.ci-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.scene-tag-bar {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  bottom: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: calc(100% - 200px);
  overflow-x: auto;
  padding: 6px 10px;
  background: color-mix(in srgb, var(--bg-card) 82%, transparent);
  backdrop-filter: blur(6px);
  border: 1px solid var(--border);
  border-radius: 12px;
  z-index: 10;
}
.scene-hint {
  position: absolute;
  right: 18px;
  top: 16px;
  font-size: 11px;
  color: var(--text-muted);
  background: color-mix(in srgb, var(--bg-card) 75%, transparent);
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid var(--border);
  z-index: 10;
}
.corner-btn {
  position: absolute;
  left: 18px;
  top: 16px;
  z-index: 10;
}
.scene-empty {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-secondary);
  text-align: center;
  z-index: 5;
}
.scene-empty.soft {
  background: color-mix(in srgb, var(--bg-primary) 55%, transparent);
}
.scene-empty h3 {
  margin: 0;
  color: var(--text-primary);
}
.scene-empty p {
  margin: 0;
  max-width: 420px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-muted);
}
</style>
