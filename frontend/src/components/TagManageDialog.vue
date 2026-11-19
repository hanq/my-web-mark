<script setup lang="ts">
/** 标签管理对话框：新增、改名、换色、删除当前主题下的标签。 */
import { ref, onMounted, computed } from 'vue'
import type { TagItem } from '../api'
import { useDataStore } from '../stores/data'
import { useThemeStore } from '../stores/theme'
import AppIcon from './AppIcon.vue'

const emit = defineEmits<{ close: [] }>()
const data = useDataStore()
const themeStore = useThemeStore()

const newName = ref('')
const newColor = ref('#4f8cff')
const err = ref('')
const busy = ref(false)
/** 正在改名的标签 id（行内编辑态） */
const editingId = ref<number | null>(null)
const editName = ref('')
const editColor = ref('')

const PALETTE = ['#4f8cff', '#38d39f', '#ffb648', '#ff5d6d', '#b26bff', '#00c2d1', '#ff8fab', '#8d9cff']

onMounted(async () => {
  if (!data.tags.length) await data.load(themeStore.activeThemeId, true)
})

const sortedTags = computed(() => [...data.tags].sort((a, b) => a.sort - b.sort || a.id - b.id))

async function create() {
  if (!newName.value.trim()) {
    err.value = '请输入标签名称'
    return
  }
  busy.value = true
  err.value = ''
  try {
    await data.addTag(themeStore.activeThemeId, newName.value.trim(), newColor.value)
    newName.value = ''
  } catch (e) {
    err.value = String(e instanceof Error ? e.message : e)
  } finally {
    busy.value = false
  }
}

function startEdit(tag: TagItem) {
  editingId.value = tag.id
  editName.value = tag.name
  editColor.value = tag.color || '#4f8cff'
}

async function saveEdit() {
  if (editingId.value == null || !editName.value.trim()) return
  busy.value = true
  try {
    await data.updateTag(editingId.value, editName.value.trim(), editColor.value)
    editingId.value = null
  } catch (e) {
    err.value = String(e instanceof Error ? e.message : e)
  } finally {
    busy.value = false
  }
}

async function remove(tag: TagItem) {
  if (!window.confirm(`删除标签「${tag.name}」？书签上的该标签关联将一并清除。`)) return
  await data.removeTag(tag.id)
}
</script>

<template>
  <div class="modal-mask" @click.self="emit('close')">
    <div class="modal">
      <h3>管理标签<span class="sub">「{{ themeStore.activeTheme?.name }}」</span></h3>

      <!-- 新增行 -->
      <div class="add-row">
        <span class="color-dot" :style="{ background: newColor }" />
        <input
          v-model="newName"
          class="input"
          placeholder="新标签名称，如：前端、待读"
          @keyup.enter="create"
        />
        <div class="palette">
          <button
            v-for="c in PALETTE"
            :key="c"
            type="button"
            class="swatch"
            :class="{ active: newColor === c }"
            :style="{ background: c }"
            @click="newColor = c"
          />
        </div>
        <button class="btn primary" :disabled="busy || !themeStore.activeThemeId" @click="create">
          <AppIcon name="plus" :size="15" /> 新增
        </button>
      </div>

      <p v-if="err" class="err">{{ err }}</p>

      <!-- 标签列表 -->
      <div class="tag-list">
        <div v-for="tag in sortedTags" :key="tag.id" class="tag-row">
          <template v-if="editingId === tag.id">
            <span class="color-dot" :style="{ background: editColor }" />
            <input v-model="editName" class="input" @keyup.enter="saveEdit" />
            <div class="palette">
              <button
                v-for="c in PALETTE"
                :key="c"
                type="button"
                class="swatch"
                :class="{ active: editColor === c }"
                :style="{ background: c }"
                @click="editColor = c"
              />
            </div>
            <button class="btn ghost icon-btn" title="保存" @click="saveEdit"><AppIcon name="check" :size="16" /></button>
            <button class="btn ghost icon-btn" title="取消" @click="editingId = null"><AppIcon name="close" :size="16" /></button>
          </template>
          <template v-else>
            <span class="color-dot" :style="{ background: tag.color }" />
            <span class="tag-name">{{ tag.name }}</span>
            <span class="tag-count">{{ data.bookmarks.filter((b) => b.tags.some((t) => t.id === tag.id)).length }} 个书签</span>
            <button class="btn ghost icon-btn" title="编辑" @click="startEdit(tag)"><AppIcon name="edit" :size="15" /></button>
            <button class="btn ghost icon-btn" title="删除" @click="remove(tag)"><AppIcon name="trash" :size="15" /></button>
          </template>
        </div>
        <div v-if="!sortedTags.length" class="empty-mini">暂无标签，先在上方新增一个</div>
      </div>

      <div class="modal-actions">
        <button class="btn primary" @click="emit('close')">完成</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sub {
  font-size: 12px;
  color: var(--text-muted);
  font-weight: 400;
  margin-left: 6px;
}
.add-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.add-row .input {
  flex: 1;
}
.color-dot {
  width: 12px;
  height: 12px;
  border-radius: 4px;
  flex-shrink: 0;
}
.palette {
  display: flex;
  gap: 4px;
}
.swatch {
  width: 14px;
  height: 14px;
  border-radius: 4px;
  border: 2px solid transparent;
  padding: 0;
  cursor: pointer;
}
.swatch.active {
  border-color: var(--text-primary);
}
.tag-list {
  max-height: 300px;
  overflow: auto;
}
.tag-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 4px;
  border-bottom: 1px dashed var(--border);
}
.tag-row .input {
  flex: 1;
}
.tag-name {
  flex: 1;
  font-size: 13px;
}
.tag-count {
  font-size: 11px;
  color: var(--text-muted);
}
.icon-btn {
  padding: 4px 6px;
  border: none;
}
.empty-mini {
  padding: 18px;
  color: var(--text-muted);
  font-size: 12px;
  text-align: center;
}
.err {
  color: var(--danger);
  font-size: 12px;
  margin: 6px 0 0;
}
</style>
