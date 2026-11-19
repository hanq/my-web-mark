import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { themeApi, type ThemeSummary, type ThemeInputModel } from '../api'
import { toMsg } from './vault'

/** 主题分组状态。 */
export const useThemeStore = defineStore('theme', () => {
  const themes = ref<ThemeSummary[]>([])
  const activeThemeId = ref<number>(0)
  const loading = ref(false)
  const error = ref('')

  const activeTheme = computed(() =>
    themes.value.find((t) => t.id === activeThemeId.value) ?? null,
  )

  async function refresh() {
    loading.value = true
    error.value = ''
    try {
      themes.value = (await themeApi.list()) ?? []
      if (activeThemeId.value && !themes.value.some((t) => t.id === activeThemeId.value)) {
        activeThemeId.value = 0
      }
    } catch (e) {
      error.value = toMsg(e)
    } finally {
      loading.value = false
    }
  }

  async function create(input: ThemeInputModel) {
    const th = await themeApi.create(input)
    await refresh()
    return th
  }

  async function update(id: number, input: ThemeInputModel) {
    await themeApi.update(id, input)
    await refresh()
  }

  async function remove(id: number) {
    await themeApi.remove(id)
    await refresh()
  }

  async function sort(ids: number[]) {
    await themeApi.sort(ids)
    await refresh()
  }

  /**
   * 侧栏拖拽重排主题：本地生效后写库，失败回滚刷新。
   * 收起态与展开态均可拖拽。
   */
  async function reorderThemes(dragId: number, targetId: number, pos: 'before' | 'after') {
    if (dragId === targetId) return
    const list = themes.value.slice()
    const from = list.findIndex((t) => t.id === dragId)
    const to = list.findIndex((t) => t.id === targetId)
    if (from < 0 || to < 0) return
    const [moved] = list.splice(from, 1)
    let to2 = list.findIndex((t) => t.id === targetId)
    if (pos === 'after') to2 += 1
    list.splice(to2, 0, moved)
    themes.value = list
    try {
      await themeApi.sort(list.map((t) => t.id))
    } catch (e) {
      error.value = toMsg(e)
      await refresh()
    }
  }

  function select(id: number) {
    activeThemeId.value = id
    localStorage.setItem('mwm-last-theme', String(id))
  }

  return { themes, activeThemeId, activeTheme, loading, error, refresh, create, update, remove, sort, select, reorderThemes }
})
