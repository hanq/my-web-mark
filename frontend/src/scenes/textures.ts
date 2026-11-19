/**
 * Canvas 纹理生成：书签卡片（玻璃拟态合成）、流光光带。
 * 卡片分层：深底 -> favicon 高斯模糊背景（正片叠底）-> 半透明磨砂玻璃层 -> 内容。
 * 均为离屏 Canvas -> CanvasTexture，避免外部图片依赖。
 */
import * as THREE from 'three'

/** 圆角矩形路径辅助。 */
function roundRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
) {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}

/** hex(#rrggbb) -> rgba 字符串；非法输入原样返回。 */
function hexToRgba(hex: string, alpha: number): string {
  const h = hex.replace('#', '')
  if (h.length !== 6) return hex
  const r = parseInt(h.slice(0, 2), 16)
  const g = parseInt(h.slice(2, 4), 16)
  const b = parseInt(h.slice(4, 6), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

/** 截断文本到指定像素宽度。 */
function fitText(ctx: CanvasRenderingContext2D, text: string, maxW: number): string {
  if (ctx.measureText(text).width <= maxW) return text
  while (text.length > 1 && ctx.measureText(text + '…').width > maxW) {
    text = text.slice(0, -1)
  }
  return text + '…'
}

export interface CardPaintOptions {
  title: string
  host: string
  color: string
  pinned?: boolean
  /** 已加载的 favicon 图片（同源）；缺省时用主题色晕染兜底 */
  faviconImg?: HTMLImageElement | null
}

const W = 384
const H = 244
const BADGE = { x: 24, y: 36, s: 56, r: 14 }

/** 绘制 favicon 高斯模糊背景：放大铺满 + blur，以正片叠底融入底色。 */
function paintBlurBackground(ctx: CanvasRenderingContext2D, opts: CardPaintOptions) {
  // 基底深色
  ctx.fillStyle = '#141a29'
  roundRect(ctx, 0, 0, W, H, 18)
  ctx.fill()

  ctx.save()
  roundRect(ctx, 0, 0, W, H, 18)
  ctx.clip()
  ctx.globalCompositeOperation = 'multiply'
  ctx.globalAlpha = 0.9
  if (opts.faviconImg && opts.faviconImg.width > 0) {
    const img = opts.faviconImg
    ctx.filter = 'blur(30px)'
    // cover 模式放大（额外放大 1.6x 保证 blur 边缘不出黑边）
    const scale = Math.max(W / img.width, H / img.height) * 1.6
    const iw = img.width * scale
    const ih = img.height * scale
    ctx.drawImage(img, (W - iw) / 2, (H - ih) / 2, iw, ih)
    ctx.filter = 'none'
  } else {
    // 无 favicon：主题色大光晕兜底
    const grad = ctx.createRadialGradient(W * 0.32, H * 0.42, 10, W * 0.32, H * 0.42, W * 0.85)
    grad.addColorStop(0, opts.color)
    grad.addColorStop(1, '#0c1120')
    ctx.fillStyle = grad
    ctx.fillRect(0, 0, W, H)
  }
  ctx.restore()
}

/** 半透明磨砂玻璃层：透出模糊背景 + 顶部高光 + 细描边。 */
function paintGlassLayer(ctx: CanvasRenderingContext2D) {
  ctx.save()
  roundRect(ctx, 0, 0, W, H, 18)
  const glass = ctx.createLinearGradient(0, 0, 0, H)
  glass.addColorStop(0, 'rgba(236, 242, 252, 0.74)')
  glass.addColorStop(1, 'rgba(214, 224, 244, 0.56)')
  ctx.fillStyle = glass
  ctx.fill()

  // 顶部高光细线（玻璃折角，弱化）
  ctx.strokeStyle = 'rgba(255,255,255,0.4)'
  ctx.lineWidth = 1.5
  roundRect(ctx, 1.5, 1.5, W - 3, H / 2.6, 16)
  ctx.stroke()

  // 外描边
  ctx.strokeStyle = 'rgba(255,255,255,0.4)'
  ctx.lineWidth = 1.5
  roundRect(ctx, 0.75, 0.75, W - 1.5, H - 1.5, 17)
  ctx.stroke()
  ctx.restore()
}

/** 内容层：徽标（favicon 或首字母）+ 标题 + 域名 + 置顶标。 */
function paintContent(ctx: CanvasRenderingContext2D, opts: CardPaintOptions) {
  // 徽标底
  ctx.fillStyle = opts.color
  roundRect(ctx, BADGE.x, BADGE.y, BADGE.s, BADGE.s, BADGE.r)
  ctx.fill()

  if (opts.faviconImg && opts.faviconImg.width > 0) {
    // favicon 白底内衬 + contain 居中
    ctx.save()
    roundRect(ctx, BADGE.x, BADGE.y, BADGE.s, BADGE.s, BADGE.r)
    ctx.fillStyle = 'rgba(255,255,255,0.94)'
    ctx.fill()
    ctx.clip()
    const pad = 7
    const s = BADGE.s - pad * 2
    const scale = Math.min(s / opts.faviconImg.width, s / opts.faviconImg.height)
    const iw = opts.faviconImg.width * scale
    const ih = opts.faviconImg.height * scale
    ctx.drawImage(
      opts.faviconImg,
      BADGE.x + (BADGE.s - iw) / 2,
      BADGE.y + (BADGE.s - ih) / 2,
      iw,
      ih,
    )
    ctx.restore()
  } else {
    ctx.fillStyle = '#ffffff'
    ctx.font = 'bold 30px "Segoe UI", sans-serif'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    const first = [...opts.title.trim()][0] ?? '?'
    ctx.fillText(first.toUpperCase(), BADGE.x + BADGE.s / 2, BADGE.y + BADGE.s / 2 + 1)
  }

  // 标题与域名（深色文字压在玻璃层上）
  ctx.textAlign = 'left'
  ctx.textBaseline = 'alphabetic'
  const maxW = W - 48
  ctx.fillStyle = '#1a2233'
  ctx.font = 'bold 26px "Segoe UI", "Microsoft YaHei", sans-serif'
  ctx.fillText(fitText(ctx, opts.title, maxW), 24, 128)
  ctx.fillStyle = '#5c687f'
  ctx.font = '20px Consolas, monospace'
  ctx.fillText(fitText(ctx, opts.host, maxW), 24, 164)

  if (opts.pinned) {
    ctx.fillStyle = '#e8962e'
    ctx.font = 'bold 22px sans-serif'
    ctx.textAlign = 'right'
    ctx.fillText('PIN', W - 24, H - 26)
  }
}

/** 完整绘制一张卡片正面到 ctx（圆角外透明）。 */
function paintCard(ctx: CanvasRenderingContext2D, opts: CardPaintOptions) {
  ctx.clearRect(0, 0, W, H)
  paintBlurBackground(ctx, opts)
  paintGlassLayer(ctx)
  paintContent(ctx, opts)

  // 边缘渐变光边：主题色细描边（弱化），顶部略亮向底部渐隐。
  // 仅作轮廓提示，配合低强度 Bloom 形成极轻微柔光，不抢内容视觉。
  ctx.save()
  const edge = ctx.createLinearGradient(0, 0, W * 0.35, H)
  edge.addColorStop(0, hexToRgba(opts.color, 0.55))
  edge.addColorStop(0.45, hexToRgba(opts.color, 0.25))
  edge.addColorStop(1, hexToRgba(opts.color, 0.08))
  ctx.strokeStyle = edge
  ctx.lineWidth = 4
  roundRect(ctx, 2, 2, W - 4, H - 4, 17)
  ctx.stroke()
  ctx.restore()
}

/**
 * 生成书签卡片正面纹理（无 favicon 初始版）。
 * favicon 到位后调用 repaintCard 整卡重绘。
 */
export function makeCardTexture(opts: CardPaintOptions): THREE.CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d')!
  paintCard(ctx, opts)
  const tex = new THREE.CanvasTexture(canvas)
  tex.colorSpace = THREE.SRGBColorSpace
  tex.anisotropy = 4
  return tex
}

/**
 * favicon 加载完成后整卡重绘（玻璃模糊背景 + 内容 + favicon 徽标）。
 * opts 为建卡时的原始参数，faviconImg 为新加载的图片。
 */
export function repaintCard(
  canvas: HTMLCanvasElement,
  texture: THREE.Texture,
  opts: CardPaintOptions,
): void {
  if (canvas.width !== W || canvas.height !== H) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  paintCard(ctx, opts)
  texture.needsUpdate = true
}

/**
 * 生成流光光带纹理：对角亮带，RepeatWrapping 供 offset 平移扫过卡片。
 * 所有卡片共享同一纹理实例。
 */
export function makeSheenTexture(): THREE.CanvasTexture {
  const S = 128
  const canvas = document.createElement('canvas')
  canvas.width = S
  canvas.height = S
  const ctx = canvas.getContext('2d')!
  const grad = ctx.createLinearGradient(0, 0, S, S)
  grad.addColorStop(0, 'rgba(255,255,255,0)')
  grad.addColorStop(0.42, 'rgba(255,255,255,0)')
  grad.addColorStop(0.5, 'rgba(255,255,255,0.85)')
  grad.addColorStop(0.58, 'rgba(255,255,255,0)')
  grad.addColorStop(1, 'rgba(255,255,255,0)')
  ctx.fillStyle = grad
  ctx.fillRect(0, 0, S, S)
  const tex = new THREE.CanvasTexture(canvas)
  tex.wrapS = THREE.RepeatWrapping
  tex.wrapT = THREE.ClampToEdgeWrapping
  tex.colorSpace = THREE.SRGBColorSpace
  return tex
}
