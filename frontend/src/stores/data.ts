import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useThemeStore } from './theme'
import {
  bookmarkApi,
  accountApi,
  browserApi,
  pageTitleApi,
  tagApi,
  faviconApi,
  type BookmarkItem,
  type AccountSummary,
  type BookmarkInputModel,
  type AccountInputModel,
  type TagItem,
} from '../api'
import { toMsg } from './vault'

/** 书签/账号/标签数据状态（按当前激活主题加载）。 */
export const useDataStore = defineStore('data', () => {
  const bookmarks = ref<BookmarkItem[]>([])
  const accounts = ref<AccountSummary[]>([])
  const tags = ref<TagItem[]>([])
  /** 全部主题的标签（2.5D 全局筛选条用） */
  const allTags = ref<TagItem[]>([])
  /** 全部主题的书签（2.5D 场景一次性加载） */
  const allBookmarks = ref<BookmarkItem[]>([])
  /** 当前筛选标签 id；null 表示显示全部 */
  const activeTagId = ref<number | null>(null)
  const themeId = ref(0)
  const loading = ref(false)
  const error = ref('')
  /** 明文密码临时缓存（复制/展示后即清），键为账号 id */
  const revealed = ref<Record<number, string>>({})
  /** favicon 缓存：书签 URL -> 同源图标路径 */
  const favicons = ref<Record<string, string>>({})
  const faviconsTried = new Set<string>()
  const faviconsInflight = new Map<string, Promise<string | null>>()

  const sortedBookmarks = computed(() =>
    [...bookmarks.value].sort((a, b) => Number(b.pinned) - Number(a.pinned)),
  )

  /** 书签/账号写操作后同步刷新主题统计（侧栏计数徽标），失败静默不影响主流程 */
  async function refreshThemeStats() {
    try {
      await useThemeStore().refresh()
    } catch {
      /* 统计刷新失败不阻塞 */
    }
  }

  /** 按当前筛选标签过滤后的书签（列表与 2.5D 场景共用）。 */
  const filteredBookmarks = computed(() => {
    if (activeTagId.value == null) return sortedBookmarks.value
    return sortedBookmarks.value.filter((b) => b.tags.some((t) => t.id === activeTagId.value))
  })

  /**
   * 拖拽重排书签：把 drag 移动到 target 前/后，写库并本地生效。
   * 筛选态下仅重排可见子集，不可见项保持原相对顺序（不破坏全局排序）。
   * 跨置顶组拖拽直接忽略（置顶组恒在前）。
   */
  async function reorderBookmarks(dragId: number, targetId: number, pos: 'before' | 'after') {
    const full = sortedBookmarks.value
    const drag = full.find((b) => b.id === dragId)
    const target = full.find((b) => b.id === targetId)
    if (!drag || !target || drag.id === target.id) return
    if (drag.pinned !== target.pinned) return // 置顶组内才可互相排序

    // 可见子集内移动
    const vis = filteredBookmarks.value.slice()
    const from = vis.findIndex((b) => b.id === dragId)
    if (from < 0) return
    const [moved] = vis.splice(from, 1)
    let to = vis.findIndex((b) => b.id === targetId)
    if (to < 0) return
    if (pos === 'after') to += 1
    vis.splice(to, 0, moved)

    // 重建全集：可见项按新顺序输出，不可见项原位保持
    const visQueue = [...vis]
    const newFull = full.map((b) => (vis.some((v) => v.id === b.id) ? visQueue.shift()! : b))

    bookmarks.value = newFull
    try {
      await bookmarkApi.sort(newFull.map((b) => b.id))
    } catch (e) {
      error.value = toMsg(e)
      await load(themeId.value, true) // 失败回滚为库内真实顺序
    }
  }

  async function load(tid: number, keepFilter = false) {
    themeId.value = tid
    if (!tid) {
      bookmarks.value = []
      accounts.value = []
      tags.value = []
      return
    }
    loading.value = true
    error.value = ''
    try {
      const [bm, acc, tg] = await Promise.all([
        bookmarkApi.listByTheme(tid),
        accountApi.listByTheme(tid),
        tagApi.listByTheme(tid),
      ])
      bookmarks.value = bm ?? []
      accounts.value = acc ?? []
      tags.value = tg ?? []
      // 同步全量书签缓存（2.5D 画廊数据源），失败静默不影响当前主题展示
      bookmarkApi
        .listAll()
        .then((all) => (allBookmarks.value = all ?? []))
        .catch(() => {})
      // 换主题后旧筛选标签不再适用；同主题刷新时保留
      if (!keepFilter && activeTagId.value != null && !tags.value.some((t) => t.id === activeTagId.value)) {
        activeTagId.value = null
      }
    } catch (e) {
      error.value = toMsg(e)
    } finally {
      loading.value = false
    }
  }

  async function loadAllTags() {
    try {
      allTags.value = (await tagApi.listAll()) ?? []
    } catch (e) {
      error.value = toMsg(e)
    }
  }

  async function loadAllBookmarks() {
    try {
      allBookmarks.value = (await bookmarkApi.listAll()) ?? []
    } catch (e) {
      error.value = toMsg(e)
    }
  }

  // --- 标签 ---

  async function addTag(themeIdVal: number, name: string, color: string) {
    const t = await tagApi.create(themeIdVal, { name, color })
    await load(themeId.value, true)
    await loadAllTags()
    return t
  }

  async function updateTag(id: number, name: string, color: string) {
    await tagApi.update(id, { name, color })
    await load(themeId.value, true)
    await loadAllTags()
  }

  async function removeTag(id: number) {
    await tagApi.remove(id)
    if (activeTagId.value === id) activeTagId.value = null
    await load(themeId.value, true)
    await loadAllTags()
  }

  function setTagFilter(id: number | null) {
    activeTagId.value = id
    localStorage.setItem('mwm-last-tag', id == null ? '' : String(id))
  }

  // --- 书签 ---

  async function addBookmark(input: BookmarkInputModel) {
    const bm = await bookmarkApi.create({ ...input, themeId: input.themeId || themeId.value })
    await load(themeId.value, true)
    await refreshThemeStats()
    return bm
  }

  async function updateBookmark(id: number, input: BookmarkInputModel) {
    await bookmarkApi.update(id, input)
    await load(themeId.value, true)
    await refreshThemeStats()
  }

  async function removeBookmark(id: number) {
    await bookmarkApi.remove(id)
    await load(themeId.value, true)
    await refreshThemeStats()
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
    await load(themeId.value, true)
    await refreshThemeStats()
    return acc
  }

  async function updateAccount(id: number, input: AccountInputModel) {
    await accountApi.update(id, input)
    delete revealed.value[id]
    await load(themeId.value, true)
    await refreshThemeStats()
  }

  async function removeAccount(id: number) {
    await accountApi.remove(id)
    delete revealed.value[id]
    await load(themeId.value, true)
    await refreshThemeStats()
  }

  /** 解锁态下查看明文密码（写入展示缓存，卡片上会展开为明文）。 */
  async function revealPassword(id: number): Promise<string> {
    const { password } = await accountApi.reveal(id)
    revealed.value[id] = password
    return password
  }

  /** 读取明文密码但不写入展示缓存（一键复制用，不触发卡片展开）。 */
  async function fetchPassword(id: number): Promise<string> {
    const { password } = await accountApi.reveal(id)
    return password
  }

  function hidePassword(id: number) {
    delete revealed.value[id]
  }

  /** 清空所有明文缓存（金库锁定/页面切换时调用）。 */
  function hideAllPasswords() {
    revealed.value = {}
  }

  /** 复制敏感文本并定时清空剪贴板。 */
  async function copySecret(text: string) {
    await browserApi.copyToClipboard(text)
    window.setTimeout(() => {
      browserApi.clearClipboard().catch(() => {})
    }, 15000)
  }

  // --- favicon ---

  /**
   * 确保书签站点图标可用；成功返回同源路径（/favicons/xxx）。
   * 同一 URL 并发去重；失败的 URL 本会话不再重试（后端另有负缓存）。
   */
  async function ensureFavicon(url: string): Promise<string | null> {
    if (favicons.value[url]) return favicons.value[url]
    if (faviconsTried.has(url)) return favicons.value[url] ?? null
    const p = faviconsInflight.get(url)
    if (p) return p
    faviconsTried.add(url)
    const task = faviconApi
      .ensure(url)
      .then((name) => {
        if (name) favicons.value[url] = `/favicons/${name}`
        return favicons.value[url] ?? null
      })
      .catch(() => null)
      .finally(() => faviconsInflight.delete(url))
    faviconsInflight.set(url, task)
    return task
  }

  /** 图标文件加载失败时移除，回退首字母显示。 */
  function faviconFailed(url: string) {
    delete favicons.value[url]
  }

  return {
    bookmarks, accounts, tags, allTags, allBookmarks, activeTagId, themeId, loading, error, revealed, favicons,
    sortedBookmarks, filteredBookmarks,
    load, loadAllTags, loadAllBookmarks,
    addTag, updateTag, removeTag, setTagFilter,
    addBookmark, updateBookmark, removeBookmark, addDroppedUrl, reorderBookmarks,
    addAccount, updateAccount, removeAccount,
    revealPassword, fetchPassword, hidePassword, hideAllPasswords, copySecret,
    ensureFavicon, faviconFailed,
  }
})
