/**
 * Wails 窗口运行时封装（无边框窗口的最小化/最大化/关闭）。
 * window.runtime 由 Wails 在 WebView 中注入；浏览器开发环境不存在，
 * 调用会抛错由 UI 忽略。
 */

interface WailsRuntime {
  WindowMinimise(): void
  WindowToggleMaximise(): void
  WindowIsMaximised(): Promise<boolean>
  Quit(): void
}

function rt(): WailsRuntime {
  const wails = (window as unknown as { runtime?: WailsRuntime }).runtime
  if (!wails) {
    throw new Error('Wails 运行时不可用（浏览器开发模式）')
  }
  return wails
}

export function minimiseWindow(): void {
  try {
    rt().WindowMinimise()
  } catch {
    /* 非 Wails 环境 */
  }
}

export function toggleMaximiseWindow(): void {
  try {
    rt().WindowToggleMaximise()
  } catch {
    /* 非 Wails 环境 */
  }
}

export async function isWindowMaximised(): Promise<boolean> {
  try {
    return await rt().WindowIsMaximised()
  } catch {
    return false
  }
}

export function quitApp(): void {
  try {
    rt().Quit()
  } catch {
    /* 非 Wails 环境 */
  }
}
