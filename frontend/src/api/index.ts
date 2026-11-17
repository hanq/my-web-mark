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

export interface BookmarkItem {
  id: number
  themeId: number
  title: string
  url: string
  favicon: string
  note: string
  pinned: boolean
  source: string
  createdAt: string
}

export interface AccountSummary {
  id: number
  themeId: number
  site: string
  username: string
  hasPwd: boolean
  url: string
  note: string
}

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
}

export const bookmarkApi = {
  listByTheme: (themeId: number) => call<BookmarkItem[]>('BookmarkService', 'ListByTheme', themeId),
  listAll: () => call<BookmarkItem[]>('BookmarkService', 'ListAll'),
  create: (input: BookmarkInputModel) => call<BookmarkItem>('BookmarkService', 'Create', input),
  update: (id: number, input: BookmarkInputModel) =>
    call<void>('BookmarkService', 'Update', id, input),
  remove: (id: number) => call<void>('BookmarkService', 'Delete', id),
  detectBrowsers: () => call<BrowserStatusModel[]>('BookmarkService', 'DetectBrowsers'),
  previewImport: (browser: string) =>
    call<ImportPreviewModel>('BookmarkService', 'PreviewImport', browser),
  executeImport: (browser: string, items: ImportPlanItemModel[]) =>
    call<ImportResultModel>('BookmarkService', 'ExecuteImport', {
      browser,
      items,
    }),
}

// ---------- Account ----------

export interface AccountInputModel {
  themeId: number
  site: string
  username: string
  password?: string
  url?: string
  note?: string
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

// ---------- Browser / Clipboard ----------

export const browserApi = {
  parseDroppedURL: (text: string) =>
    call<ParsedURLModel>('BrowserService', 'ParseDroppedURL', text),
  openInDefaultBrowser: (url: string) =>
    call<void>('BrowserService', 'OpenInDefaultBrowser', { url }),
  copyToClipboard: (text: string) => call<void>('BrowserService', 'CopyToClipboard', { text }),
  clearClipboard: () => call<void>('BrowserService', 'ClearClipboard'),
}

export const pageTitleApi = {
  fetch: (url: string) => call<string>('PageTitleService', 'FetchTitle', url),
}
