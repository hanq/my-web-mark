/**
 * Canvas 纹理生成：书签卡片正反面、主题岛名牌。
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

export interface CardTextureOptions {
  title: string
  host: string
  color: string // 主题色
  textColor?: string
  pinned?: boolean
}

/**
 * 生成书签卡片正面纹理（标题 + 域名 + 主题色条）。
 * 卡片世界尺寸约 3 x 1.9，用 384x244 px 纹理。
 */
export function makeCardTexture(opts: CardTextureOptions): THREE.CanvasTexture {
  const W = 384
  const H = 244
  const canvas = document.createElement('canvas')
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d')!

  // 底色
  ctx.fillStyle = opts.textColor ?? '#1c2333'
  roundRect(ctx, 0, 0, W, H, 18)
  ctx.fill()

  // 主题色顶条
  ctx.fillStyle = opts.color
  roundRect(ctx, 0, 0, W, 10, 5)
  ctx.fill()

  // 站点首字母徽标
  ctx.fillStyle = opts.color
  roundRect(ctx, 24, 36, 56, 56, 14)
  ctx.fill()
  ctx.fillStyle = '#ffffff'
  ctx.font = 'bold 30px "Segoe UI", sans-serif'
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  const first = [...opts.title.trim()][0] ?? '?'
  ctx.fillText(first.toUpperCase(), 52, 65)

  // 标题（最多两行，自动截断）
  ctx.fillStyle = '#e8ecf4'
  ctx.font = 'bold 26px "Segoe UI", "Microsoft YaHei", sans-serif'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'alphabetic'
  const maxW = W - 48
  let title = opts.title
  if (ctx.measureText(title).width > maxW) {
    while (title.length > 1 && ctx.measureText(title + '…').width > maxW) {
      title = title.slice(0, -1)
    }
    title += '…'
  }
  ctx.fillText(title, 24, 128)

  // 域名
  ctx.fillStyle = '#8b94a8'
  ctx.font = '20px Consolas, monospace'
  let host = opts.host
  if (ctx.measureText(host).width > maxW) {
    while (host.length > 1 && ctx.measureText(host + '…').width > maxW) {
      host = host.slice(0, -1)
    }
    host += '…'
  }
  ctx.fillText(host, 24, 164)

  // 置顶标记
  if (opts.pinned) {
    ctx.fillStyle = '#ffb648'
    ctx.font = 'bold 22px sans-serif'
    ctx.textAlign = 'right'
    ctx.fillText('PIN', W - 24, H - 26)
  }

  const tex = new THREE.CanvasTexture(canvas)
  tex.colorSpace = THREE.SRGBColorSpace
  tex.anisotropy = 4
  return tex
}

/**
 * 生成卡片背面纹理（暗色 + 图案），点击/悬停时可见。
 */
export function makeCardBackTexture(color: string): THREE.CanvasTexture {
  const W = 384
  const H = 244
  const canvas = document.createElement('canvas')
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d')!

  ctx.fillStyle = '#141a29'
  roundRect(ctx, 0, 0, W, H, 18)
  ctx.fill()

  ctx.strokeStyle = color
  ctx.globalAlpha = 0.5
  ctx.lineWidth = 3
  roundRect(ctx, 12, 12, W - 24, H - 24, 12)
  ctx.stroke()

  ctx.globalAlpha = 0.85
  ctx.fillStyle = color
  ctx.font = 'bold 34px "Segoe UI", sans-serif'
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText('OPEN', W / 2, H / 2)

  const tex = new THREE.CanvasTexture(canvas)
  tex.colorSpace = THREE.SRGBColorSpace
  return tex
}

/**
 * 生成主题岛名牌纹理（主题名 + 数量）。
 */
export function makeLabelTexture(name: string, color: string, count: number): THREE.CanvasTexture {
  const W = 512
  const H = 128
  const canvas = document.createElement('canvas')
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d')!

  ctx.fillStyle = 'rgba(0,0,0,0)'
  ctx.clearRect(0, 0, W, H)

  // 名牌底
  ctx.fillStyle = 'rgba(20,26,41,0.92)'
  roundRect(ctx, 4, 4, W - 8, H - 8, 20)
  ctx.fill()
  ctx.strokeStyle = color
  ctx.lineWidth = 4
  roundRect(ctx, 4, 4, W - 8, H - 8, 20)
  ctx.stroke()

  // 左侧色块
  ctx.fillStyle = color
  roundRect(ctx, 24, 30, 12, H - 60, 6)
  ctx.fill()

  // 主题名
  ctx.fillStyle = '#e8ecf4'
  ctx.font = 'bold 44px "Segoe UI", "Microsoft YaHei", sans-serif'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'middle'
  let name2 = name
  const maxW = W - 150
  if (ctx.measureText(name2).width > maxW) {
    while (name2.length > 1 && ctx.measureText(name2 + '…').width > maxW) {
      name2 = name2.slice(0, -1)
    }
    name2 += '…'
  }
  ctx.fillText(name2, 56, H / 2 - 8)

  // 数量
  ctx.fillStyle = '#8b94a8'
  ctx.font = '24px "Segoe UI", sans-serif'
  ctx.textAlign = 'right'
  ctx.fillText(`${count} 条`, W - 28, H / 2 - 6)

  const tex = new THREE.CanvasTexture(canvas)
  tex.colorSpace = THREE.SRGBColorSpace
  return tex
}
