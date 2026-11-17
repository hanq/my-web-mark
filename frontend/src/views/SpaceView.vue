<script setup lang="ts">
/**
 * 2.5D 空间视图：Three.js 场景为主界面，主题岛台 + 书签卡片。
 * 数据变化时重建场景；点击书签调用系统浏览器打开。
 */
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import * as THREE from 'three'
import { SpaceEngine } from '../scenes/engine'
import { useThemeStore } from '../stores/theme'
import { useDataStore } from '../stores/data'
import { useSkinStore } from '../stores/skin'
import { browserApi, bookmarkApi, type BookmarkItem } from '../api'
import AppIcon from '../components/AppIcon.vue'
import BookmarkDialog from '../components/BookmarkDialog.vue'
import ImportWizard from '../components/ImportWizard.vue'

const themeStore = useThemeStore()
const data = useDataStore()
const skin = useSkinStore()

const containerEl = ref<HTMLDivElement | null>(null)
const engine = ref<SpaceEngine | null>(null)
const hoverText = ref('')
const hoverPos = ref({ x: 0, y: 0 })
const showBookmarkDialog = ref(false)
const newBookmarkUrl = ref('')
const showImport = ref(false)
const toast = ref('')
let toastTimer = 0

const hasSceneData = computed(
  () => themeStore.themes.some((t) => t.bookmarkCount > 0),
)

function flash(msg: string, isErr = false) {
  toast.value = msg
  toastType.value = isErr ? 'error' : ''
  window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => (toast.value = ''), 2200)
}
const toastType = ref('')

/** 拉取全部数据并重建场景。 */
async function rebuild() {
  await themeStore.refresh()
  const all = (await bookmarkApi.listAll()) ?? []
  engine.value?.build(themeStore.themes, all)
}

onMounted(async () => {
  if (!containerEl.value) return
  engine.value = new SpaceEngine(containerEl.value, {
    onBookmarkClick: openBookmark,
    onThemeClick: (id) => {
      themeStore.select(id)
      data.load(id)
    },
    onHover: (text, x, y) => {
      hoverText.value = text ?? ''
      hoverPos.value = { x, y }
    },
  })
  // 皮肤联动
  skin.registerSceneSync((s) => engine.value?.applySkin(s))
  await rebuild()
})

onUnmounted(() => {
  engine.value?.dispose()
  engine.value = null
})

async function openBookmark(bm: BookmarkItem) {
  try {
    await browserApi.openInDefaultBrowser(bm.url)
  } catch (e) {
    flash(String(e instanceof Error ? e.message : e), true)
  }
}

// 主题/书签数据变化时刷新场景（如列表视图增删后切回）
watch(
  () => themeStore.themes.length,
  () => rebuild(),
)

// 拖拽 URL 到 3D 场景快速入库
async function onDrop(e: DragEvent) {
  const dt = e.dataTransfer
  if (!dt) return
  const text =
    dt.getData('text/uri-list') || dt.getData('text/html') || dt.getData('text/plain')
  if (!text) return
  e.preventDefault()
  try {
    const parsed = await browserApi.parseDroppedURL(text)
    newBookmarkUrl.value = parsed.rawUrl
    showBookmarkDialog.value = true
  } catch (err) {
    flash(String(err instanceof Error ? err.message : err), true)
  }
}

onUnmounted(() => window.clearTimeout(toastTimer))
// 保持 THREE 引用（tree-shaking 保险 + 未来扩展）
void THREE
</script>

<template>
  <div class="space-view" @dragover.prevent @drop.prevent="onDrop">
    <div ref="containerEl" class="scene-container" />

    <!-- 悬浮提示 -->
    <div
      v-if="hoverText"
      class="hover-tip"
      :style="{ left: hoverPos.x + 14 + 'px', top: hoverPos.y + 14 + 'px' }"
    >
      {{ hoverText }}
    </div>

    <!-- 左下角操作 -->
    <div class="corner-actions">
      <button class="btn" title="复位视角 (Esc)" @click="engine?.resetView()">
        <AppIcon name="refresh" :size="15" /> 复位视角
      </button>
      <button class="btn primary" @click="showBookmarkDialog = true">
        <AppIcon name="plus" :size="15" /> 新书签
      </button>
      <button class="btn" @click="showImport = true">
        <AppIcon name="download" :size="15" /> 导入
      </button>
    </div>

    <!-- 操作提示 -->
    <div class="scene-hint">滚轮缩放 · 拖拽平移 · 点击卡片打开 · 点击岛台聚焦 · Esc 复位</div>

    <!-- 空场景引导 -->
    <div v-if="!hasSceneData" class="scene-empty">
      <AppIcon name="cube" :size="46" />
      <h3>你的书签空间还是空的</h3>
      <p>新建主题并添加书签，或从浏览器一键导入，它们会以 2.5D 岛台的形式呈现在这里。</p>
      <button class="btn primary" @click="showImport = true">
        <AppIcon name="download" :size="15" /> 从浏览器导入
      </button>
    </div>

    <BookmarkDialog
      v-if="showBookmarkDialog"
      @close="showBookmarkDialog = false; newBookmarkUrl = ''; rebuild()"
    />
    <ImportWizard v-if="showImport" @close="showImport = false" @done="rebuild()" />

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
.hover-tip {
  position: fixed;
  max-width: 340px;
  padding: 6px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  box-shadow: var(--shadow-card);
  font-size: 12px;
  color: var(--text-primary);
  pointer-events: none;
  z-index: 30;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.corner-actions {
  position: absolute;
  left: 18px;
  bottom: 18px;
  display: flex;
  gap: 10px;
  z-index: 10;
}
.scene-hint {
  position: absolute;
  right: 18px;
  bottom: 18px;
  font-size: 11px;
  color: var(--text-muted);
  background: color-mix(in srgb, var(--bg-card) 75%, transparent);
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid var(--border);
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
  pointer-events: auto;
  z-index: 5;
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
