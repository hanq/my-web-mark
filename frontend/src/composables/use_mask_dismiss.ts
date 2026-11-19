/**
 * 遮罩点击关闭判定。
 *
 * 问题：在对话框内按下选中文本、拖到遮罩上松开时，浏览器派发的 click
 * 目标是弹窗与遮罩的共同祖先（即遮罩自身），配合 .self 修饰符会误判为
 * "点击遮罩"导致对话框意外关闭。
 *
 * 方案：记录 mousedown 坐标；click 时校验按下与松开位置几乎重合
 * （视为真实点击）才允许关闭。模板用法：
 *   <div class="modal-mask" @mousedown="onDown" @click.self="onUp">
 */
export function useMaskDismiss(onDismiss: () => void) {
  let downX = 0
  let downY = 0

  /** mousedown 阶段记录按下坐标（监听在整个遮罩上，含弹窗内部）。 */
  function onDown(e: MouseEvent) {
    downX = e.clientX
    downY = e.clientY
  }

  /** click.self 阶段判定：位移在阈值内才触发关闭。 */
  function onUp(e: MouseEvent) {
    const moved = Math.abs(e.clientX - downX) + Math.abs(e.clientY - downY)
    if (moved <= 6) onDismiss()
  }

  return { onDown, onUp }
}
