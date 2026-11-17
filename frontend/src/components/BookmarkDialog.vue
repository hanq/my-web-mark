<script setup lang="ts">
/** 书签新增/编辑对话框。 */
import { ref, computed } from 'vue'
import type { BookmarkItem } from '../api'
import { useDataStore } from '../stores/data'
import { useThemeStore } from '../stores/theme'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ editing?: BookmarkItem | null; presetThemeId?: number }>()
const emit = defineEmits<{ close: [] }>()

const data = useDataStore()
const themeStore = useThemeStore()

const themeId = ref(props.editing?.themeId ?? props.presetThemeId ?? themeStore.activeThemeId)
const title = ref(props.editing?.title ?? '')
const url = ref(props.editing?.url ?? '')
const note = ref(props.editing?.note ?? '')
const pinned = ref(props.editing?.pinned ?? false)
const err = ref('')
const busy = ref(false)

const isEdit = computed(() => !!props.editing)

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
