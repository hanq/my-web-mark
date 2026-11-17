import { defineStore } from 'pinia'
import { ref } from 'vue'
import { vaultApi, type VaultStatusModel } from '../api'

/** 金库（管理员密码）状态与解锁流程。 */
export const useVaultStore = defineStore('vault', () => {
  const setupRequired = ref(true)
  const unlocked = ref(false)
  const busy = ref(false)
  const error = ref('')

  async function refresh() {
    try {
      const st: VaultStatusModel = await vaultApi.status()
      setupRequired.value = st.setupRequired
      unlocked.value = st.unlocked
    } catch (e) {
      error.value = String(e)
    }
  }

  async function setup(password: string) {
    busy.value = true
    error.value = ''
    try {
      await vaultApi.setup(password)
      setupRequired.value = false
      unlocked.value = true
    } catch (e) {
      error.value = toMsg(e)
      throw e
    } finally {
      busy.value = false
    }
  }

  async function unlock(password: string) {
    busy.value = true
    error.value = ''
    try {
      await vaultApi.unlock(password)
      unlocked.value = true
    } catch (e) {
      error.value = toMsg(e)
      throw e
    } finally {
      busy.value = false
    }
  }

  async function lock() {
    await vaultApi.lock()
    unlocked.value = false
  }

  async function changePassword(oldPwd: string, newPwd: string) {
    busy.value = true
    error.value = ''
    try {
      await vaultApi.changePassword(oldPwd, newPwd)
    } catch (e) {
      error.value = toMsg(e)
      throw e
    } finally {
      busy.value = false
    }
  }

  return { setupRequired, unlocked, busy, error, refresh, setup, unlock, lock, changePassword }
})

export function toMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}
