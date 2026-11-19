<script setup lang="ts">
/**
 * 金库解锁 / 首次设置管理员密码对话框。
 * mode=setup 首次设置；mode=unlock 解锁；不可关闭（数据受保护）。
 */
import { ref, computed } from 'vue'
import { useVaultStore } from '../stores/vault'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ mode: 'setup' | 'unlock' }>()
const emit = defineEmits<{ close: [] }>()
const vault = useVaultStore()

const pwd = ref('')
const pwd2 = ref('')
const showPwd = ref(false)
const localErr = ref('')

const isSetup = computed(() => props.mode === 'setup')

async function submit() {
  localErr.value = ''
  if (!pwd.value) {
    localErr.value = '请输入密码'
    return
  }
  if (isSetup.value) {
    if (pwd.value.length < 6) {
      localErr.value = '密码至少 6 位'
      return
    }
    if (pwd.value !== pwd2.value) {
      localErr.value = '两次输入不一致'
      return
    }
    try {
      await vault.setup(pwd.value)
    } catch {
      /* vault.error 已带信息 */
    }
    return
  }
  try {
    await vault.unlock(pwd.value)
  } catch {
    /* keep dialog */
  }
}

const errMsg = computed(() => localErr.value || vault.error)
</script>

<template>
  <div class="modal-mask">
    <div class="modal vault-modal">
      <button v-if="!isSetup" class="btn ghost close-x" title="取消" @click="emit('close')">
        <AppIcon name="close" :size="16" />
      </button>
      <div class="vault-head">
        <span class="vault-icon"><AppIcon :name="isSetup ? 'shield' : 'key'" :size="26" /></span>
        <h3>{{ isSetup ? '设置管理员密码' : '输入管理员密码' }}</h3>
        <p class="vault-tip">
          {{
            isSetup
              ? '管理员密码用于加密账号密码数据，设置后请牢记。密码不会明文存储，无法找回。'
              : '解锁后可查看与复制账号密码；锁定状态下敏感信息不可见。'
          }}
        </p>
      </div>

      <div class="field">
        <label>管理员密码</label>
        <div class="pwd-row">
          <input
            v-model="pwd"
            :type="showPwd ? 'text' : 'password'"
            class="input"
            placeholder="至少 6 位"
            autocomplete="off"
            @keyup.enter="submit"
          />
          <button class="btn ghost pwd-eye" type="button" @click="showPwd = !showPwd">
            <AppIcon :name="showPwd ? 'eyeOff' : 'eye'" />
          </button>
        </div>
      </div>

      <div v-if="isSetup" class="field">
        <label>确认密码</label>
        <input
          v-model="pwd2"
          type="password"
          class="input"
          placeholder="再次输入"
          autocomplete="off"
          @keyup.enter="submit"
        />
      </div>

      <p v-if="errMsg" class="vault-err">{{ errMsg }}</p>

      <button class="btn primary vault-btn" :disabled="vault.busy" @click="submit">
        <AppIcon name="unlock" />
        {{ vault.busy ? '处理中…' : isSetup ? '设置并解锁' : '解锁' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.vault-modal {
  width: min(420px, calc(100vw - 48px));
  position: relative;
}
.close-x {
  position: absolute;
  top: 12px;
  right: 12px;
}
.vault-head {
  text-align: center;
  margin-bottom: 20px;
}
.vault-icon {
  display: inline-flex;
  padding: 14px;
  border-radius: 16px;
  background: var(--accent-soft);
  color: var(--accent);
  margin-bottom: 12px;
}
.vault-head h3 {
  margin: 0 0 8px;
}
.vault-tip {
  margin: 0;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.6;
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
.vault-err {
  color: var(--danger);
  font-size: 12px;
  margin: 4px 0 0;
}
.vault-btn {
  width: 100%;
  justify-content: center;
  margin-top: 16px;
  padding: 10px;
}
</style>
