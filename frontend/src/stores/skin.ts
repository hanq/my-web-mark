import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeSkin =
  | 'dark'
  | 'light'
  | 'neon'
  | 'chalkboard'
  | 'obsidian'
  | 'bamboo'
  | 'jade'
  | 'rainbow'

const STORAGE_KEY = 'mwm-skin'

/** 全部合法皮肤（localStorage 回读校验用） */
const VALID_SKINS: ThemeSkin[] = [
  'dark', 'light', 'neon', 'chalkboard', 'obsidian', 'bamboo', 'jade', 'rainbow',
]

/** 界面皮肤（data-theme 换肤）状态；切换时写 <html data-theme> 并同步 3D 场景回调。 */
export const useSkinStore = defineStore('skin', () => {
  const skin = ref<ThemeSkin>(load())
  /** 3D 场景配色联动回调（由场景引擎注册） */
  let sceneSync: ((skin: ThemeSkin) => void) | null = null

  function load(): ThemeSkin {
    const v = localStorage.getItem(STORAGE_KEY)
    return VALID_SKINS.includes(v as ThemeSkin) ? (v as ThemeSkin) : 'dark'
  }

  function apply() {
    document.documentElement.dataset.theme = skin.value
    sceneSync?.(skin.value)
  }

  function setSkin(s: ThemeSkin) {
    skin.value = s
    localStorage.setItem(STORAGE_KEY, s)
  }

  function registerSceneSync(fn: (skin: ThemeSkin) => void) {
    sceneSync = fn
    fn(skin.value)
  }

  // 响应 skin 变化应用样式
  watch(skin, apply, { immediate: true })

  return { skin, setSkin, registerSceneSync }
})
