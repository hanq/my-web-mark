<script setup lang="ts">
/** 设置页：管理员密码管理 + 界面皮肤。 */
import { ref, computed } from 'vue'
import { useVaultStore } from '../stores/vault'
import { useSkinStore, type ThemeSkin } from '../stores/skin'
import AppIcon from '../components/AppIcon.vue'
import VaultLock from '../components/VaultLock.vue'

const vault = useVaultStore()
const skin = useSkinStore()

// 修改密码表单
const oldPwd = ref('')
const newPwd = ref('')
const newPwd2 = ref('')
const changeErr = ref('')
const changeOk = ref(false)
const changing = ref(false)

const canChange = computed(() => vault.unlocked)

async function doChange() {
  changeErr.value = ''
  changeOk.value = false
  if (!oldPwd.value || !newPwd.value) {
    changeErr.value = '请填写完整'
    return
  }
  if (newPwd.value !== newPwd2.value) {
    changeErr.value = '两次输入的新密码不一致'
    return
  }
  changing.value = true
  try {
    await vault.changePassword(oldPwd.value, newPwd.value)
    changeOk.value = true
    oldPwd.value = newPwd.value = newPwd2.value = ''
  } catch (e) {
    changeErr.value = String(e instanceof Error ? e.message : e)
  } finally {
    changing.value = false
  }
}

const skins: { key: ThemeSkin; label: string; desc: string }[] = [
  { key: 'dark', label: '暗夜科技', desc: '深蓝灰底色，科技蓝点缀' },
  { key: 'light', label: '极简白', desc: '明亮简洁，日间舒适' },
  { key: 'neon', label: '赛博霓虹', desc: '紫霓虹辉光，赛博风格' },
]
</script>

<template>
  <div class="settings-view">
    <div class="settings-col">
      <!-- 管理员密码 -->
      <div class="card section">
        <h3><AppIcon name="shield" /> 管理员密码</h3>
        <div class="status-line">
          <span>金库状态：</span>
          <span class="badge" :class="vault.unlocked ? 'success' : 'danger'">
            {{ vault.unlocked ? '已解锁' : '已锁定' }}
          </span>
          <button v-if="vault.unlocked" class="btn ghost" @click="vault.lock()">
            <AppIcon name="lock" :size="14" /> 立即锁定
          </button>
        </div>

        <template v-if="canChange">
          <div class="field">
            <label>当前密码</label>
            <input v-model="oldPwd" type="password" class="input" autocomplete="off" />
          </div>
          <div class="grid-2">
            <div class="field">
              <label>新密码</label>
              <input v-model="newPwd" type="password" class="input" autocomplete="off" />
            </div>
            <div class="field">
              <label>确认新密码</label>
              <input v-model="newPwd2" type="password" class="input" autocomplete="off" />
            </div>
          </div>
          <p v-if="changeErr" class="msg err">{{ changeErr }}</p>
          <p v-if="changeOk" class="msg ok">密码已修改，全部账号密文已用新密钥重加密。</p>
          <button class="btn primary" :disabled="changing" @click="doChange">
            <AppIcon name="key" :size="15" /> {{ changing ? '处理中…' : '修改管理员密码' }}
          </button>
          <p class="hint">修改密码会用新密钥重加密所有账号密码，过程在本地完成。</p>
        </template>
        <p v-else class="hint">解锁金库后可修改管理员密码。</p>
      </div>

      <!-- 界面皮肤 -->
      <div class="card section">
        <h3><AppIcon name="palette" /> 界面皮肤</h3>
        <div class="skin-grid">
          <button
            v-for="s in skins"
            :key="s.key"
            class="skin-card"
            :class="{ active: skin.skin === s.key }"
            :data-skin="s.key"
            @click="skin.setSkin(s.key)"
          >
            <span class="skin-preview" />
            <span class="skin-label">{{ s.label }}</span>
            <span class="skin-desc">{{ s.desc }}</span>
          </button>
        </div>
      </div>

      <!-- 关于 -->
      <div class="card section">
        <h3><AppIcon name="cube" /> 关于</h3>
        <p class="hint">
          My Web Mark v1.0 · 本地书签与账号管理工具<br />
          数据存储于本地 SQLite 数据库；密码使用 argon2id + AES-256-GCM 加密。<br />
          数据目录：%APPDATA%/my-web-mark
        </p>
      </div>
    </div>

    <!-- 首次设置引导（复用金库对话框） -->
    <VaultLock v-if="vault.setupRequired" mode="setup" />
  </div>
</template>

<style scoped>
.settings-view {
  height: 100%;
  overflow: auto;
  display: flex;
  justify-content: center;
  padding: 26px 20px;
}
.settings-col {
  width: min(680px, 100%);
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.section {
  padding: 20px;
}
.section h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 14px;
  font-size: 15px;
}
.status-line {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  font-size: 13px;
  color: var(--text-secondary);
}
.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.msg {
  font-size: 12px;
  margin: 4px 0;
}
.msg.err {
  color: var(--danger);
}
.msg.ok {
  color: var(--success);
}
.hint {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.7;
  margin: 10px 0 0;
}
.skin-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}
.skin-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  padding: 14px;
  border: 1.5px solid var(--border);
  border-radius: 12px;
  background: var(--bg-input);
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}
.skin-card:hover {
  border-color: var(--border-strong);
}
.skin-card.active {
  border-color: var(--accent);
  background: var(--accent-soft);
}
.skin-preview {
  width: 100%;
  height: 44px;
  border-radius: 8px;
}
.skin-card[data-skin='dark'] .skin-preview {
  background: linear-gradient(135deg, #10141f 40%, #4f8cff);
}
.skin-card[data-skin='light'] .skin-preview {
  background: linear-gradient(135deg, #f0f3f9 40%, #3b6fe0);
}
.skin-card[data-skin='neon'] .skin-preview {
  background: linear-gradient(135deg, #12122a 40%, #b26bff, #00e5b0);
}
.skin-label {
  font-size: 13px;
  color: var(--text-primary);
}
.skin-desc {
  font-size: 11px;
  color: var(--text-muted);
}
</style>
