/**
 * Go 服务调用封装。
 * Wails 在 wailsjs/go/ 下生成绑定；此处统一封装 window.go 调用并给出 TS 类型。
 * 方法名与 Go 绑定服务（app 包各 *Service）一一对应。
 */

/* eslint-disable @typescript-eslint/no-explicit-any */
type WailsService = Record<string, (...args: any[]) => Promise<any>>

function call<T>(service: string, method: string, ...args: unknown[]): Promise<T> {
  const svc = (window as any).go?.app?.[service] as WailsService | undefined
  if (!svc || typeof svc[method] !== 'function') {
    return Promise.reject(new Error(`后端服务不可用: ${service}.${method}`))
  }
  return svc[method](...args) as Promise<T>
}

// ---------- 数据模型 ----------

export interface ThemeItem {
  id: number
  name: string
  color: string
  icon: string
  sort: number
  createdAt: string
}

export interface ThemeSummary extends ThemeItem {
  bookmarkCount: number
  accountCount: number
}

export interface TagItem {
  id: number
  themeId: number
  name: string
  color: string
  sort: number
}

export interface BookmarkItem {
  id: number
  themeId: number
  title: string
  url: string
  favicon: string
  note: string
  pinned: boolean
  source: string
  tags: TagItem[]
  createdAt: string
}

export interface AccountSummary {
  id: number
  themeId: number
  kind: AccountKind
  bookmarkId: number | null
  site: string
  username: string
  hasPwd: boolean
  url: string
  note: string
  tags: TagItem[]
}

/** 账号类型：网站账号、设备/开机密码、WiFi 密码、其他凭据。 */
export type AccountKind = 'web' | 'device' | 'wifi' | 'other'

export interface VaultStatusModel {
  setupRequired: boolean
  unlocked: boolean
}

export interface BrowserStatusModel {
  browser: string
  name: string
  running: boolean
  installed: boolean
  bookmarks: number
  error?: string
}

export interface BrowserBookmarkModel {
  title: string
  url: string
  folder: string
}

export interface ImportPreviewModel {
  browser: string
  items: BrowserBookmarkModel[]
  folders: string[]
}

export interface ImportPlanItemModel {
  folder: string
  targetThemeId: number
}

export interface ImportResultModel {
  imported: number
  skipped: number
  errors?: string[]
}

export interface ParsedURLModel {
  rawUrl: string
  title: string
  host: string
}

// ---------- Vault ----------

export const vaultApi = {
  status: () => call<VaultStatusModel>('VaultService', 'Status'),
  setup: (password: string) => call<void>('VaultService', 'Setup', password),
  unlock: (password: string) => call<void>('VaultService', 'Unlock', { password }),
  lock: () => call<void>('VaultService', 'Lock'),
  changePassword: (oldPassword: string, newPassword: string) =>
    call<void>('VaultService', 'ChangePassword', { oldPassword, newPassword }),
}

// ---------- Theme ----------

export interface ThemeInputModel {
  name: string
  color: string
  icon: string
}

export const themeApi = {
  list: () => call<ThemeSummary[]>('ThemeService', 'List'),
  create: (input: ThemeInputModel) => call<ThemeItem>('ThemeService', 'Create', input),
  update: (id: number, input: ThemeInputModel) => call<void>('ThemeService', 'Update', id, input),
  sort: (ids: number[]) => call<void>('ThemeService', 'Sort', { ids }),
  remove: (id: number) => call<void>('ThemeService', 'Delete', id),
}

// ---------- Bookmark ----------

export interface BookmarkInputModel {
  themeId: number
  title: string
  url: string
  favicon?: string
  note?: string
  pinned?: boolean
  tagIds?: number[]
}

export const bookmarkApi = {
  listByTheme: (themeId: number) => call<BookmarkItem[]>('BookmarkService', 'ListByTheme', themeId),
  listAll: () => call<BookmarkItem[]>('BookmarkService', 'ListAll'),
  create: (input: BookmarkInputModel) => call<BookmarkItem>('BookmarkService', 'Create', input),
  update: (id: number, input: BookmarkInputModel) =>
    call<void>('BookmarkService', 'Update', id, input),
  remove: (id: number) => call<void>('BookmarkService', 'Delete', id),
  /** 批量保存拖拽排序（按数组顺序） */
  sort: (ids: number[]) => call<void>('BookmarkService', 'Sort', { ids }),
  detectBrowsers: () => call<BrowserStatusModel[]>('BookmarkService', 'DetectBrowsers'),
  previewImport: (browser: string) =>
    call<ImportPreviewModel>('BookmarkService', 'PreviewImport', browser),
  executeImport: (browser: string, items: ImportPlanItemModel[]) =>
    call<ImportResultModel>('BookmarkService', 'ExecuteImport', {
      browser,
      items,
    }),
}

// ---------- Tag ----------

export interface TagInputModel {
  name: string
  color: string
}

export const tagApi = {
  listByTheme: (themeId: number) => call<TagItem[]>('TagService', 'ListByTheme', themeId),
  listAll: () => call<TagItem[]>('TagService', 'ListAll'),
  create: (themeId: number, input: TagInputModel) =>
    call<TagItem>('TagService', 'Create', themeId, input),
  update: (id: number, input: TagInputModel) => call<void>('TagService', 'Update', id, input),
  remove: (id: number) => call<void>('TagService', 'Delete', id),
}

// ---------- Favicon ----------

export const faviconApi = {
  /** 确保站点图标已缓存，返回缓存文件名（空串表示不可用，前端回退首字母） */
  ensure: (url: string) => call<string>('FaviconService', 'Ensure', url),
}

// ---------- Account ----------

export interface AccountInputModel {
  themeId: number
  kind?: AccountKind
  bookmarkId?: number
  site: string
  username: string
  password?: string
  url?: string
  note?: string
  tagIds?: number[]
}

export const accountApi = {
  listByTheme: (themeId: number) =>
    call<AccountSummary[]>('AccountService', 'ListByTheme', themeId),
  create: (input: AccountInputModel) => call<AccountSummary>('AccountService', 'Create', input),
  update: (id: number, input: AccountInputModel) =>
    call<void>('AccountService', 'Update', id, input),
  remove: (id: number) => call<void>('AccountService', 'Delete', id),
  reveal: (id: number) => call<{ id: number; password: string }>('AccountService', 'Reveal', id),
}

// ---------- Export / Import ----------

export interface ImportThemeResultModel {
  themeName: string
  themeId: number
  imported: number
  skipped: number
  tagCount: number
}

export const exportApi = {
  /** 导出主题书签；返回文件路径，空串表示用户取消 */
  exportTheme: (themeId: number, format: 'html' | 'json') =>
    call<string>('ExportService', 'ExportTheme', themeId, format),
  /** 打开文件导入为主题；null 表示用户取消 */
  importTheme: () => call<ImportThemeResultModel | null>('ExportService', 'ImportTheme'),
}

// ---------- Settings ----------

export const settingsApi = {
  get: (key: string) => call<string>('SettingsService', 'Get', key),
  set: (key: string, value: string) => call<void>('SettingsService', 'Set', key, value),
}

/** 设置键常量（与 Go 端 SettingKeyLaunchBrowser 对应） */
export const SETTING_LAUNCH_BROWSER = 'launch_browser'

// ---------- Browser / Clipboard ----------

export const browserApi = {
  parseDroppedURL: (text: string) =>
    call<ParsedURLModel>('BrowserService', 'ParseDroppedURL', text),
  /** 按用户偏好打开 URL；返回回退提示（空串=正常） */
  openInDefaultBrowser: (url: string) =>
    call<string>('BrowserService', 'OpenInDefaultBrowser', { url }),
  copyToClipboard: (text: string) => call<void>('BrowserService', 'CopyToClipboard', { text }),
  clearClipboard: () => call<void>('BrowserService', 'ClearClipboard'),
}

export const pageTitleApi = {
  fetch: (url: string) => call<string>('PageTitleService', 'FetchTitle', url),
}
