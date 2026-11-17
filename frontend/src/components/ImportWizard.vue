<script setup lang="ts">
/** 浏览器书签导入向导：检测 -> 预览 -> 选目标主题 -> 执行导入。 */
import { ref, computed, onMounted } from 'vue'
import {
  bookmarkApi,
  type BrowserStatusModel,
  type ImportPreviewModel,
  type ImportPlanItemModel,
  type ImportResultModel,
} from '../api'
import { useThemeStore } from '../stores/theme'
import AppIcon from './AppIcon.vue'

const emit = defineEmits<{ close: []; done: [] }>()
const themeStore = useThemeStore()

type Step = 'detect' | 'plan' | 'result'

const step = ref<Step>('detect')
const browsers = ref<BrowserStatusModel[]>([])
const preview = ref<ImportPreviewModel | null>(null)
const plan = ref<ImportPlanItemModel[]>([])
const result = ref<ImportResultModel | null>(null)
const selected = ref('')
const loading = ref(false)
const err = ref('')

onMounted(refresh)

async function refresh() {
  loading.value = true
  err.value = ''
  try {
    browsers.value = (await bookmarkApi.detectBrowsers()) ?? []
  } catch (e) {
    err.value = String(e instanceof Error ? e.message : e)
  } finally {
    loading.value = false
  }
}

/** 已安装且可读取的浏览器才能继续 */
const available = computed(() => browsers.value.filter((b) => b.installed && b.bookmarks >= 0))

async function pick(browser: string) {
  selected.value = browser
  loading.value = true
  err.value = ''
  try {
    preview.value = await bookmarkApi.previewImport(browser)
    // 默认计划：每个顶层文件夹自动建同名主题（targetThemeId=0）
    plan.value = (preview.value?.folders ?? []).map((f) => ({ folder: f, targetThemeId: 0 }))
    step.value = 'plan'
  } catch (e) {
    err.value = String(e instanceof Error ? e.message : e)
  } finally {
    loading.value = false
  }
}

function targetOf(folder: string): number {
  return plan.value.find((p) => p.folder === folder)?.targetThemeId ?? 0
}

function setTarget(folder: string, id: number) {
  const item = plan.value.find((p) => p.folder === folder)
  if (item) item.targetThemeId = id
}

const totalPlanned = computed(() =>
  (preview.value?.items ?? []).filter((i) => targetOf(firstSeg(i.folder)) !== 0).length,
)

function firstSeg(folder: string): string {
  if (!folder) return ''
  const i = folder.indexOf('/')
  return i >= 0 ? folder.slice(0, i) : folder
}

async function runImport() {
  loading.value = true
  err.value = ''
  try {
    result.value = await bookmarkApi.executeImport(selected.value, plan.value)
    step.value = 'result'
  } catch (e) {
    err.value = String(e instanceof Error ? e.message : e)
  } finally {
    loading.value = false
  }
}

function finish() {
  emit('done')
  emit('close')
}
</script>

<template>
  <div class="modal-mask" @click.self="emit('close')">
    <div class="modal import-modal">
      <!-- 第一步：检测浏览器 -->
      <template v-if="step === 'detect'">
        <h3>从浏览器导入书签</h3>
        <p class="hint">选择一个浏览器读取其本地书签（只读，不修改浏览器数据）。</p>

        <div v-if="loading" class="loading-row"><AppIcon name="refresh" /> 检测中…</div>
        <template v-else>
          <div
            v-for="b in browsers"
            :key="b.browser"
            class="browser-row"
            :class="{ disabled: !b.installed || b.bookmarks < 0 }"
            @click="b.installed && b.bookmarks >= 0 && pick(b.browser)"
          >
            <span class="browser-icon"><AppIcon name="chrome" /></span>
            <div class="browser-info">
              <div class="browser-name">
                {{ b.name }}
                <span v-if="b.running" class="badge success">运行中</span>
                <span v-if="b.installed && b.bookmarks >= 0" class="badge">{{ b.bookmarks }} 条书签</span>
              </div>
              <div v-if="b.error" class="browser-err">{{ b.error }}</div>
            </div>
            <AppIcon v-if="b.installed && b.bookmarks >= 0" name="external" class="go-icon" />
          </div>
          <p v-if="!available.length && !loading" class="hint">未检测到可导入的浏览器数据。</p>
        </template>
        <p v-if="err" class="err">{{ err }}</p>
      </template>

      <!-- 第二步：选择导入计划 -->
      <template v-else-if="step === 'plan' && preview">
        <h3>导入「{{ browsers.find((b) => b.browser === selected)?.name }}」</h3>
        <p class="hint">共 {{ preview.items.length }} 条书签。为每个顶层文件夹选择目标主题：</p>

        <div class="folder-list">
          <div v-for="f in preview.folders" :key="f" class="folder-row">
            <span class="folder-name">
              <AppIcon name="folder" :size="15" />
              {{ f }}
            </span>
            <select class="input target-select" :value="targetOf(f)" @change="setTarget(f, +($event.target as HTMLSelectElement).value)">
              <option :value="0">自动创建同名主题</option>
              <option v-for="t in themeStore.themes" :key="t.id" :value="t.id">导入到「{{ t.name }}」</option>
            </select>
          </div>
          <div v-if="!preview.folders.length" class="folder-row">
            <span class="folder-name">（无文件夹分组）</span>
            <span class="badge muted">将导入到「导入」主题</span>
          </div>
        </div>

        <p class="hint small">计划导入 {{ totalPlanned }} 条 / 共 {{ preview.items.length }} 条（重复 URL 自动跳过）</p>

        <div class="modal-actions">
          <button class="btn ghost" @click="step = 'detect'">上一步</button>
          <button class="btn primary" :disabled="loading" @click="runImport">
            <AppIcon name="download" />
            {{ loading ? '导入中…' : '开始导入' }}
          </button>
        </div>
        <p v-if="err" class="err">{{ err }}</p>
      </template>

      <!-- 第三步：结果 -->
      <template v-else-if="step === 'result' && result">
        <h3>导入完成</h3>
        <div class="result-grid">
          <div class="result-item ok">
            <span class="num">{{ result.imported }}</span>
            <span>导入成功</span>
          </div>
          <div class="result-item skip">
            <span class="num">{{ result.skipped }}</span>
            <span>重复跳过</span>
          </div>
          <div class="result-item" :class="{ bad: result.errors?.length }">
            <span class="num">{{ result.errors?.length ?? 0 }}</span>
            <span>失败</span>
          </div>
        </div>
        <ul v-if="result.errors?.length" class="err-list">
          <li v-for="(e, i) in result.errors.slice(0, 5)" :key="i">{{ e }}</li>
        </ul>
        <div class="modal-actions">
          <button class="btn primary" @click="finish">完成</button>
        </div>
      </template>

      <button class="btn ghost close-x" @click="emit('close')"><AppIcon name="close" /></button>
    </div>
  </div>
</template>

<style scoped>
.import-modal {
  position: relative;
  width: min(560px, calc(100vw - 48px));
}
.close-x {
  position: absolute;
  top: 14px;
  right: 14px;
}
.hint {
  color: var(--text-muted);
  font-size: 12px;
  margin: 0 0 14px;
}
.hint.small {
  margin: 10px 0 0;
}
.loading-row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  padding: 20px 0;
}
.browser-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  margin-bottom: 10px;
  cursor: pointer;
  transition: all 0.15s;
}
.browser-row:hover:not(.disabled) {
  border-color: var(--accent);
  background: var(--accent-soft);
}
.browser-row.disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.browser-icon {
  color: var(--accent);
  display: flex;
}
.browser-info {
  flex: 1;
}
.browser-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.browser-err {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 3px;
}
.go-icon {
  color: var(--text-muted);
}
.folder-list {
  max-height: 300px;
  overflow: auto;
}
.folder-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px dashed var(--border);
}
.folder-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.target-select {
  width: 200px;
  flex-shrink: 0;
  padding: 6px 10px;
}
.result-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin: 16px 0;
}
.result-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 16px;
  border-radius: 10px;
  background: var(--bg-input);
  font-size: 12px;
  color: var(--text-secondary);
}
.result-item.ok .num {
  color: var(--success);
}
.result-item.skip .num {
  color: var(--warning);
}
.result-item.bad .num {
  color: var(--danger);
}
.result-item .num {
  font-size: 24px;
  font-weight: 700;
}
.err-list {
  margin: 8px 0 0;
  padding-left: 18px;
  color: var(--danger);
  font-size: 12px;
  max-height: 120px;
  overflow: auto;
}
.err {
  color: var(--danger);
  font-size: 12px;
  margin: 8px 0 0;
}
</style>
