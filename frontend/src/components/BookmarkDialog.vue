<script setup lang="ts">
/** 书签新增/编辑对话框。 */
import { ref, computed, onMounted } from 'vue'
import type { BookmarkItem, TagItem } from '../api'
import { useDataStore } from '../stores/data'
import { useThemeStore } from '../stores/theme'
import AppIcon from './AppIcon.vue'

const props = defineProps<{
  editing?: BookmarkItem | null
  presetThemeId?: number
  /** 拖拽入库预填 URL（新增场景） */
  presetUrl?: string
  /** 拖拽入库预填标题（新增场景） */
  presetTitle?: string
}>()
const emit = defineEmits<{ close: [] }>()

const data = useDataStore()
const themeStore = useThemeStore()

const themeId = ref(props.editing?.themeId ?? props.presetThemeId ?? themeStore.activeThemeId)
const title = ref(props.editing?.title ?? props.presetTitle ?? '')
const url = ref(props.editing?.url ?? props.presetUrl ?? '')
const note = ref(props.editing?.note ?? '')
const pinned = ref(props.editing?.pinned ?? false)
const selectedTagIds = ref<number[]>(props.editing?.tags.map((t) => t.id) ?? [])
const err = ref('')
const busy = ref(false)

const isEdit = computed(() => !!props.editing)

// 主题可切换时标签列表跟随主题变化
const themeTags = computed(() => {
  const tid = themeId.value
  if (tid === themeStore.activeThemeId) return data.tags
  // 非当前主题：从全局标签中过滤（编辑态切主题时保持可选）
  return data.allTags.filter((t) => t.themeId === tid)
})
onMounted(() => {
  data.loadAllTags()
})

async function quickAddTag() {
  const name = quickTagName.value.trim()
  if (!name) return
  try {
    const t: TagItem = await data.addTag(themeId.value, name, '#8d9cff')
    selectedTagIds.value.push(t.id)
    quickTagName.value = ''
  } catch (e) {
    err.value = String(e instanceof Error ? e.message : e)
  }
}
const quickTagName = ref('')

function toggleTag(id: number) {
  const i = selectedTagIds.value.indexOf(id)
  if (i >= 0) selectedTagIds.value.splice(i, 1)
  else selectedTagIds.value.push(id)
}

async function save() {
  if (!url.value.trim()) {
    err.value = '请输入 URL'
    return
  }
  if (!themeId.value) {
    err.value = '请选择主题'
    return
  }
  busy.value = true
  err.value = ''
  try {
    const input = {
      themeId: themeId.value,
      title: title.value.trim(),
      url: url.value.trim(),
      note: note.value.trim(),
      pinned: pinned.value,
      tagIds: selectedTagIds.value,
    }
    if (props.editing) await data.updateBookmark(props.editing.id, input)
    else await data.addBookmark(input)
    emit('close')
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
      <h3>{{ isEdit ? '编辑书签' : '新增书签' }}</h3>

      <div class="field">
        <label>所属主题</label>
        <select v-model="themeId" class="input">
          <option v-for="t in themeStore.themes" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
      </div>

      <div class="field">
        <label>URL</label>
        <input v-model="url" class="input" placeholder="https:// 或裸域名均可" @keyup.enter="save" />
      </div>

      <div class="field">
        <label>标题（留空自动使用 URL）</label>
        <input v-model="title" class="input" placeholder="书签标题" @keyup.enter="save" />
      </div>

      <div class="field">
        <label>备注</label>
        <textarea v-model="note" class="input" rows="2" placeholder="可选" />
      </div>

      <div class="field">
        <label>标签（可多选，用于分类筛选）</label>
        <div class="tag-select">
          <button
            v-for="t in themeTags"
            :key="t.id"
            type="button"
            class="chip"
            :class="{ active: selectedTagIds.includes(t.id) }"
            @click="toggleTag(t.id)"
          >
            <span class="chip-dot" :style="{ background: t.color || 'var(--accent)' }" />
            {{ t.name }}
            <AppIcon v-if="selectedTagIds.includes(t.id)" name="check" :size="13" />
          </button>
          <span v-if="!themeTags.length" class="no-tags">该主题暂无标签</span>
        </div>
        <div class="quick-add">
          <input v-model="quickTagName" class="input" placeholder="快速新建标签" @keyup.enter="quickAddTag" />
          <button class="btn ghost" type="button" @click="quickAddTag">
            <AppIcon name="plus" :size="15" /> 添加
          </button>
        </div>
      </div>

      <label class="pin-row">
        <input v-model="pinned" type="checkbox" />
        <AppIcon name="pin" :size="15" />
        置顶显示
      </label>

      <p v-if="err" class="err">{{ err }}</p>

      <div class="modal-actions">
        <button class="btn ghost" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="busy" @click="save">
          <AppIcon name="check" />
          保存
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tag-select {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--bg-input);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.12s;
}
.chip:hover {
  border-color: var(--accent);
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
.no-tags {
  font-size: 12px;
  color: var(--text-muted);
}
.quick-add {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}
.quick-add .input {
  flex: 1;
}
.pin-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
  user-select: none;
}
.pin-row input {
  accent-color: var(--accent);
}
.err {
  color: var(--danger);
  font-size: 12px;
  margin: 4px 0 0;
}
</style>
