import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  bookmarkApi,
  accountApi,
  browserApi,
  pageTitleApi,
  type BookmarkItem,
  type AccountSummary,
  type BookmarkInputModel,
  type AccountInputModel,
} from '../api'
import { toMsg } from './vault'

/** 书签与账号数据状态（按当前激活主题加载）。 */
export const useDataStore = defineStore('data', () => {
  const bookmarks = ref<BookmarkItem[]>([])
  const accounts = ref<AccountSummary[]>([])
  const themeId = ref(0)
  const loading = ref(false)
  const error = ref('')
  /** 明文密码临时缓存（复制/展示后即清），键为账号 id */
  const revealed = ref<Record<number, string>>({})

  const sortedBookmarks = computed(() =>
    [...bookmarks.value].sort((a, b) => Number(b.pinned) - Number(a.pinned)),
  )

  async function load(tid: number) {
    themeId.value = tid
    if (!tid) {
      bookmarks.value = []
      accounts.value = []
      return
    }
    loading.value = true
    error.value = ''
    try {
      const [bm, acc] = await Promise.all([
        bookmarkApi.listByTheme(tid),
        accountApi.listByTheme(tid),
      ])
      bookmarks.value = bm ?? []
      accounts.value = acc ?? []
    } catch (e) {
      error.value = toMsg(e)
    } finally {
      loading.value = false
    }
  }

  // --- 书签 ---

  async function addBookmark(input: BookmarkInputModel) {
    const bm = await bookmarkApi.create({ ...input, themeId: input.themeId || themeId.value })
    await load(themeId.value)
    return bm
  }

  async function updateBookmark(id: number, input: BookmarkInputModel) {
    await bookmarkApi.update(id, input)
    await load(themeId.value)
  }

  async function removeBookmark(id: number) {
    await bookmarkApi.remove(id)
    await load(themeId.value)
  }

  /** 拖拽 URL 快速入库：解析 -> 抓标题 -> 创建。 */
  async function addDroppedUrl(text: string, tid?: number) {
    const parsed = await browserApi.parseDroppedURL(text)
    let title = parsed.title
    try {
      const fetched = await pageTitleApi.fetch(parsed.rawUrl)
      if (fetched) title = fetched
    } catch {
      /* 抓取失败用 host */
    }
    return addBookmark({
      themeId: tid ?? themeId.value,
      title,
      url: parsed.rawUrl,
    })
  }

  // --- 账号 ---

  async function addAccount(input: AccountInputModel) {
    const acc = await accountApi.create({ ...input, themeId: input.themeId || themeId.value })
    revealed.value = {}
    await load(themeId.value)
    return acc
  }

  async function updateAccount(id: number, input: AccountInputModel) {
    await accountApi.update(id, input)
    delete revealed.value[id]
    await load(themeId.value)
  }

  async function removeAccount(id: number) {
    await accountApi.remove(id)
    delete revealed.value[id]
    await load(themeId.value)
  }

  /** 解锁态下查看明文密码。 */
  async function revealPassword(id: number): Promise<string> {
    const { password } = await accountApi.reveal(id)
    revealed.value[id] = password
    return password
  }

  function hidePassword(id: number) {
    delete revealed.value[id]
  }

  /** 复制敏感文本并定时清空剪贴板。 */
  async function copySecret(text: string) {
    await browserApi.copyToClipboard(text)
    window.setTimeout(() => {
      browserApi.clearClipboard().catch(() => {})
    }, 15000)
  }

  return {
    bookmarks, accounts, themeId, loading, error, revealed,
    sortedBookmarks,
    load,
    addBookmark, updateBookmark, removeBookmark, addDroppedUrl,
    addAccount, updateAccount, removeAccount,
    revealPassword, hidePassword, copySecret,
  }
})
