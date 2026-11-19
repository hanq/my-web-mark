<script setup lang="ts">
/**
 * 账号新增/编辑对话框。
 * 除网站账号外还支持设备/开机密码、WiFi 密码等非网页凭据：
 * 站点名、用户名、密码、网址、备注均非必填（至少填一项），可关联当前主题的书签自动带出名称与网址。
 */
import { ref, computed, watch, onMounted } from 'vue'
import { bookmarkApi, tagApi, type AccountSummary, type AccountKind, type BookmarkItem, type TagItem } from '../api'
import { useDataStore } from '../stores/data'
import { useThemeStore } from '../stores/theme'
import { useVaultStore } from '../stores/vault'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ editing?: AccountSummary | null; presetThemeId?: number }>()
const emit = defineEmits<{ close: [] }>()

const data = useDataStore()
const themeStore = useThemeStore()
const vault = useVaultStore()

const KINDS: { key: AccountKind; label: string; icon: string; hint: string }[] = [
  { key: 'web', label: '网站', icon: 'globe', hint: '网站登录账号' },
  { key: 'device', label: '设备', icon: 'monitor', hint: '电脑/手机/路由器等设备密码' },
  { key: 'wifi', label: 'WiFi', icon: 'wifi', hint: 'WiFi 连接密码' },
  { key: 'other', label: '其他', icon: 'key', hint: '其他类型的凭据' },
]

const themeId = ref(props.editing?.themeId ?? props.presetThemeId ?? themeStore.activeThemeId)
const kind = ref<AccountKind>(props.editing?.kind ?? 'web')
const bookmarkId = ref<number>(props.editing?.bookmarkId ?? 0)
const site = ref(props.editing?.site ?? '')
const username = ref(props.editing?.username ?? '')
const password = ref('')
const url = ref(props.editing?.url ?? '')
const note = ref(props.editing?.note ?? '')
const tagIds = ref<number[]>(props.editing?.tags?.map((t) => t.id) ?? [])
const showPwd = ref(false)
const err = ref('')
const busy = ref(false)

/** 当前主题下的书签候选（用于关联） */
const bookmarks = ref<BookmarkItem[]>([])
/** 当前主题下的标签候选 */
const tagOptions = ref<TagItem[]>([])

const isEdit = computed(() => !!props.editing)
const kindHint = computed(() => KINDS.find((k) => k.key === kind.value)?.hint ?? '')
const siteLabel = computed(() =>
  kind.value === 'web' ? '站点名称' : kind.value === 'wifi' ? 'WiFi 名称 (SSID)' : kind.value === 'device' ? '设备名称' : '名称',
)
const sitePlaceholder = computed(() =>
  kind.value === 'web'
    ? '如：GitHub（可留空，关联书签时自动带出）'
    : kind.value === 'wifi'
      ? '如：Home-5G'
      : kind.value === 'device'
        ? '如：办公电脑 / 路由器'
        : '如：门禁密码',
)
const userLabel = computed(() => (kind.value === 'wifi' ? '账号（一般留空）' : kind.value === 'device' ? '用户名（可选）' : '用户名（可选）'))
const pwdLabel = computed(() =>
  kind.value === 'wifi' ? 'WiFi 密码' : kind.value === 'device' ? '设备密码' : '密码',
)

async function loadOptions() {
  if (!themeId.value) return
  try {
    const [bms, tgs] = await Promise.all([
      bookmarkApi.listByTheme(themeId.value),
      tagApi.listByTheme(themeId.value),
    ])
    bookmarks.value = bms ?? []
    tagOptions.value = tgs ?? []
    // 主题切换后剔除不属于新主题的标签
    const allowed = new Set(tagOptions.value.map((t) => t.id))
    tagIds.value = tagIds.value.filter((id) => allowed.has(id))
    if (bookmarkId.value && !bookmarks.value.some((b) => b.id === bookmarkId.value)) {
      bookmarkId.value = 0
    }
  } catch {
    bookmarks.value = []
    tagOptions.value = []
  }
}

onMounted(loadOptions)
watch(themeId, loadOptions)

/** 选择关联书签时，用书签标题/网址补齐空字段 */
function onBookmarkChange() {
  const bm = bookmarks.value.find((b) => b.id === bookmarkId.value)
  if (!bm) return
  if (!site.value.trim()) site.value = bm.title
  if (!url.value.trim()) url.value = bm.url
}

function toggleTag(id: number) {
  const i = tagIds.value.indexOf(id)
  if (i >= 0) tagIds.value.splice(i, 1)
  else tagIds.value.push(id)
}

function switchKind(k: AccountKind) {
  kind.value = k
}

async function save() {
  if (!themeId.value) {
    err.value = '请选择主题'
    return
  }
  const filled = [site.value, username.value, password.value, url.value, note.value].some((v) => v.trim())
  if (!filled) {
    err.value = '站点、用户名、密码、网址、备注至少填写一项'
    return
  }
  if (password.value && !vault.unlocked) {
    // 加密需要密钥：在当前对话框上直接解锁，无需退出重进
    const ok = await vault.requestUnlock('保存密码需要先解锁金库')
    if (!ok) return
  }
  busy.value = true
  err.value = ''
  try {
    const input = {
      themeId: themeId.value,
      kind: kind.value,
      bookmarkId: bookmarkId.value || undefined,
      site: site.value.trim(),
      username: username.value.trim(),
      password: password.value || undefined,
      url: url.value.trim(),
      note: note.value.trim(),
      tagIds: tagIds.value,
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
    <div class="modal acc-modal">
      <h3>{{ isEdit ? '编辑条目' : '新增条目' }}</h3>

      <div class="kind-tabs">
        <button
          v-for="k in KINDS"
          :key="k.key"
          type="button"
          class="kind-tab"
          :class="{ active: kind === k.key }"
          :title="k.hint"
          @click="switchKind(k.key)"
        >
          <AppIcon :name="k.icon" :size="16" />
          <span>{{ k.label }}</span>
        </button>
      </div>
      <p class="kind-hint">{{ kindHint }}</p>

      <div class="field">
        <label>所属主题</label>
        <select v-model="themeId" class="input">
          <option v-for="t in themeStore.themes" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
      </div>

      <div class="field">
        <label>关联书签（可选）</label>
        <select v-model="bookmarkId" class="input" @change="onBookmarkChange">
          <option :value="0">不关联</option>
          <option v-for="b in bookmarks" :key="b.id" :value="b.id">{{ b.title || b.url }}</option>
        </select>
        <span class="field-tip">关联后会自动带出书签名称与网址</span>
      </div>

      <div class="grid-2">
        <div class="field">
          <label>{{ siteLabel }}</label>
          <input v-model="site" class="input" :placeholder="sitePlaceholder" />
        </div>
        <div class="field">
          <label>网址（可选）</label>
          <input v-model="url" class="input" placeholder="https://" />
        </div>
      </div>

      <div class="grid-2">
        <div class="field">
          <label>{{ userLabel }}</label>
          <input v-model="username" class="input" autocomplete="off" :placeholder="kind === 'wifi' ? '可留空' : ''" />
        </div>
        <div class="field">
          <label>{{ pwdLabel }}{{ isEdit ? '（留空表示不修改）' : '（可留空）' }}</label>
          <div class="pwd-row">
            <input
              v-model="password"
              :type="showPwd ? 'text' : 'password'"
              class="input"
              :placeholder="isEdit ? '不修改则留空' : '可留空，稍后再补'"
              autocomplete="new-password"
            />
            <button class="btn ghost pwd-eye" type="button" @click="showPwd = !showPwd">
              <AppIcon :name="showPwd ? 'eyeOff' : 'eye'" />
            </button>
          </div>
        </div>
      </div>

      <div class="field">
        <label>标签（可选，用于区分不同主题/分类）</label>
        <div v-if="tagOptions.length" class="tag-pick">
          <button
            v-for="t in tagOptions"
            :key="t.id"
            type="button"
            class="tag-chip"
            :class="{ on: tagIds.includes(t.id) }"
            :style="tagIds.includes(t.id) ? { borderColor: t.color, color: t.color } : undefined"
            @click="toggleTag(t.id)"
          >
            <span class="tag-dot" :style="{ background: t.color }" />
            {{ t.name }}
          </button>
        </div>
        <span v-else class="field-tip">该主题还没有标签，可先在书签页新建标签</span>
      </div>

      <div class="field">
        <label>备注</label>
        <textarea v-model="note" class="input" rows="2" placeholder="可选，如：开机密码 / 家里 WiFi" />
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
.acc-modal {
  max-height: 86vh;
  overflow-y: auto;
}
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
.field-tip {
  font-size: 11px;
  color: var(--text-dim);
  margin-top: 4px;
}
.kind-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 4px;
}
.kind-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 6px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-dim);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}
.kind-tab:hover {
  color: var(--text);
  border-color: var(--accent);
}
.kind-tab.active {
  color: var(--accent);
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 12%, transparent);
}
.kind-hint {
  font-size: 11px;
  color: var(--text-dim);
  margin: 0 0 10px;
}
.tag-pick {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-dim);
  font-size: 12px;
  cursor: pointer;
}
.tag-chip.on {
  background: color-mix(in srgb, currentColor 12%, transparent);
}
.tag-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}
</style>
