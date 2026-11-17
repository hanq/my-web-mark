<script setup lang="ts">
/** 主题（分组）新建/编辑对话框。 */
import { ref, computed } from 'vue'
import type { ThemeSummary } from '../api'
import { useThemeStore } from '../stores/theme'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ editing?: ThemeSummary | null }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const themeStore = useThemeStore()

const name = ref(props.editing?.name ?? '')
const color = ref(props.editing?.color ?? '#4f8cff')

const PALETTE = [
  '#4f8cff', '#38d39f', '#ffb648', '#ff5d6d',
  '#b26bff', '#00c2d1', '#ff8fab', '#8d9cff',
]

const err = ref('')
const busy = ref(false)
const isEdit = computed(() => !!props.editing)

async function save() {
  if (!name.value.trim()) {
    err.value = '请输入主题名称'
    return
  }
  busy.value = true
  err.value = ''
  try {
    const input = { name: name.value.trim(), color: color.value, icon: '' }
    if (props.editing) {
      await themeStore.update(props.editing.id, input)
    } else {
      await themeStore.create(input)
    }
    emit('saved')
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
      <h3>{{ isEdit ? '编辑主题' : '新建主题' }}</h3>

      <div class="field">
        <label>主题名称</label>
        <input v-model="name" class="input" placeholder="如：开发、设计、影音" @keyup.enter="save" />
      </div>

      <div class="field">
        <label>主题色</label>
        <div class="color-row">
          <button
            v-for="c in PALETTE"
            :key="c"
            type="button"
            class="color-dot"
            :class="{ active: color === c }"
            :style="{ background: c }"
            @click="color = c"
          />
          <input v-model="color" type="color" class="color-picker" title="自定义颜色" />
        </div>
      </div>

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
.color-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.color-dot {
  width: 26px;
  height: 26px;
  border-radius: 8px;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 0.12s;
}
.color-dot:hover {
  transform: scale(1.12);
}
.color-dot.active {
  border-color: var(--text-primary);
  box-shadow: 0 0 0 2px var(--bg-card) inset;
}
.color-picker {
  width: 34px;
  height: 26px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: none;
  cursor: pointer;
}
.err {
  color: var(--danger);
  font-size: 12px;
  margin: 4px 0 0;
}
</style>
