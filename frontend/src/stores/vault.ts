import { defineStore } from 'pinia'
import { ref } from 'vue'
import { vaultApi, type VaultStatusModel } from '../api'

/** 金库（管理员密码）状态与解锁流程。 */
export const useVaultStore = defineStore('vault', () => {
  const setupRequired = ref(true)
  const unlocked = ref(false)
  const busy = ref(false)
  const error = ref('')
  /** 待处理解锁请求的提示文案；非空时界面弹出解锁对话框。 */
  const unlockRequest = ref('')
  // 发起解锁请求的操作（如查看密码）挂起的 Promise 回调
  let unlockResolver: ((ok: boolean) => void) | null = null

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
      settleUnlock(true)
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
      settleUnlock(true)
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

  /**
   * 在任意位置请求解锁：已解锁立即返回 true；否则弹出解锁对话框，
   * 等待用户在对话框内解锁（true）或取消（false）。
   * 这样查看/复制密码无需先退出当前页面再解锁。
   */
  function requestUnlock(reason = '该操作需要先解锁金库'): Promise<boolean> {
    if (unlocked.value) return Promise.resolve(true)
    // 重复请求时让先前的等待直接作废
    settleUnlock(false)
    unlockRequest.value = reason
    return new Promise<boolean>((resolve) => {
      unlockResolver = resolve
    })
  }

  /** 结束挂起的解锁请求：ok=true 表示已解锁，false 表示用户取消。 */
  function settleUnlock(ok: boolean) {
    const fn = unlockResolver
    unlockResolver = null
    unlockRequest.value = ''
    if (fn) fn(ok)
  }

  return {
    setupRequired,
    unlocked,
    busy,
    error,
    unlockRequest,
    refresh,
    setup,
    unlock,
    lock,
    changePassword,
    requestUnlock,
    settleUnlock,
  }
})

export function toMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}
