<script setup lang="ts">
/** 账号新增/编辑对话框。 */
import { ref, computed } from 'vue'
import type { AccountSummary } from '../api'
import { useDataStore } from '../stores/data'
import { useThemeStore } from '../stores/theme'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ editing?: AccountSummary | null; presetThemeId?: number }>()
const emit = defineEmits<{ close: [] }>()

const data = useDataStore()
const themeStore = useThemeStore()

const themeId = ref(props.editing?.themeId ?? props.presetThemeId ?? themeStore.activeThemeId)
const site = ref(props.editing?.site ?? '')
const username = ref(props.editing?.username ?? '')
const password = ref('')
const url = ref(props.editing?.url ?? '')
const note = ref(props.editing?.note ?? '')
const showPwd = ref(false)
const err = ref('')
const busy = ref(false)

const isEdit = computed(() => !!props.editing)

async function save() {
  if (!site.value.trim()) {
    err.value = '请输入站点名称'
    return
  }
  if (!username.value.trim()) {
    err.value = '请输入用户名'
    return
  }
  if (!isEdit.value && !password.value) {
    err.value = '请输入密码'
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
      site: site.value.trim(),
      username: username.value.trim(),
      password: password.value || undefined,
      url: url.value.trim(),
      note: note.value.trim(),
    }
    if (props.editing) await data.updateAccount(props.editing.id, input)
    else await data.addAccount(input)
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
      <h3>{{ isEdit ? '编辑账号' : '新增账号' }}</h3>

      <div class="field">
        <label>所属主题</label>
        <select v-model="themeId" class="input">
          <option v-for="t in themeStore.themes" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
      </div>

      <div class="grid-2">
        <div class="field">
          <label>站点</label>
          <input v-model="site" class="input" placeholder="如：GitHub" />
        </div>
        <div class="field">
          <label>网址（可选）</label>
          <input v-model="url" class="input" placeholder="https://" />
        </div>
      </div>

      <div class="field">
        <label>用户名</label>
        <input v-model="username" class="input" autocomplete="off" />
      </div>

      <div class="field">
        <label>密码{{ isEdit ? '（留空表示不修改）' : '' }}</label>
        <div class="pwd-row">
          <input
            v-model="password"
            :type="showPwd ? 'text' : 'password'"
            class="input"
            :placeholder="isEdit ? '不修改则留空' : '账号密码'"
            autocomplete="new-password"
          />
          <button class="btn ghost pwd-eye" type="button" @click="showPwd = !showPwd">
            <AppIcon :name="showPwd ? 'eyeOff' : 'eye'" />
          </button>
        </div>
      </div>

      <div class="field">
        <label>备注</label>
        <textarea v-model="note" class="input" rows="2" placeholder="可选" />
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
.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.pwd-row {
  display: flex;
  gap: 8px;
}
.pwd-row .input {
  flex: 1;
}
.pwd-eye {
  flex-shrink: 0;
}
.err {
  color: var(--danger);
  font-size: 12px;
  margin: 4px 0 0;
}
</style>
