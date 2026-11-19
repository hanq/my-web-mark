<script setup lang="ts">
/** 主题书签导出对话框：选择格式（HTML 浏览器通用 / JSON 完整备份）。 */
import { ref, computed } from 'vue'
import { exportApi } from '../api'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ themeId: number; themeName: string }>()
const emit = defineEmits<{ close: [] }>()

const format = ref<'html' | 'json'>('html')
const busy = ref(false)
const resultPath = ref('')
const err = ref('')
const cancelled = ref(false)

const formats = computed(() => [
  {
    key: 'html' as const,
    name: 'HTML 书签文件',
    desc: 'Netscape 标准格式，可导入 Edge / Chrome / Firefox 等浏览器；含备注，标签映射为文件夹',
    icon: 'external',
  },
  {
    key: 'json' as const,
    name: 'JSON 完整备份',
    desc: '本应用专有格式，完整保留标签、备注、置顶与主题色，适合备份与迁移',
    icon: 'shield',
  },
])

async function doExport() {
  busy.value = true
  err.value = ''
  cancelled.value = false
  try {
    const path = await exportApi.exportTheme(props.themeId, format.value)
    if (!path) {
      cancelled.value = true
    } else {
      resultPath.value = path
    }
  } catch (e) {
    err.value = String(e instanceof Error ? e.message : e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="modal-mask" @click.self="emit('close')">
    <div class="modal">
      <h3>导出「{{ props.themeName }}」</h3>

      <template v-if="!resultPath && !cancelled">
        <div class="fmt-list">
          <button
            v-for="f in formats"
            :key="f.key"
            type="button"
            class="fmt-item"
            :class="{ active: format === f.key }"
            @click="format = f.key"
          >
            <AppIcon :name="f.icon" :size="18" class="fmt-icon" />
            <span class="fmt-text">
              <span class="fmt-name">{{ f.name }}</span>
              <span class="fmt-desc">{{ f.desc }}</span>
            </span>
            <AppIcon v-if="format === f.key" name="check" :size="16" class="fmt-check" />
          </button>
        </div>

        <p v-if="err" class="err">{{ err }}</p>
        <div class="modal-actions">
          <button class="btn ghost" @click="emit('close')">取消</button>
          <button class="btn primary" :disabled="busy" @click="doExport">
            <AppIcon name="download" :size="15" />
            {{ busy ? '导出中…' : '选择位置并导出' }}
          </button>
        </div>
      </template>

      <template v-else-if="cancelled">
        <p class="hint">已取消导出。</p>
        <div class="modal-actions">
          <button class="btn primary" @click="emit('close')">好的</button>
        </div>
      </template>

      <template v-else>
        <div class="done-row">
          <AppIcon name="check" :size="18" class="done-icon" />
          <div class="done-text">
            <div class="done-title">导出成功</div>
            <div class="done-path" :title="resultPath">{{ resultPath }}</div>
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn primary" @click="emit('close')">完成</button>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.fmt-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.fmt-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1.5px solid var(--border);
  border-radius: 10px;
  background: var(--bg-input);
  cursor: pointer;
  text-align: left;
  transition: all 0.12s;
}
.fmt-item:hover {
  border-color: var(--border-strong);
}
.fmt-item.active {
  border-color: var(--accent);
  background: var(--accent-soft);
}
.fmt-icon {
  color: var(--text-secondary);
  flex-shrink: 0;
}
.fmt-item.active .fmt-icon {
  color: var(--accent);
}
.fmt-text {
  display: flex;
  flex-direction: column;
  gap: 3px;
  flex: 1;
  min-width: 0;
}
.fmt-name {
  font-size: 13px;
  color: var(--text-primary);
}
.fmt-desc {
  font-size: 11px;
  color: var(--text-muted);
  line-height: 1.5;
}
.fmt-check {
  color: var(--accent);
  flex-shrink: 0;
}
.err {
  color: var(--danger);
  font-size: 12px;
  margin: 10px 0 0;
}
.done-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.done-icon {
  color: var(--success);
  margin-top: 2px;
}
.done-title {
  font-size: 14px;
  color: var(--text-primary);
  margin-bottom: 4px;
}
.done-path {
  font-size: 12px;
  color: var(--text-muted);
  word-break: break-all;
}
.hint {
  font-size: 12px;
  color: var(--text-muted);
}
</style>
