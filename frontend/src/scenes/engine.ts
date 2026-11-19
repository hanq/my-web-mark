/**
 * 3D 卡片画廊引擎（Cover Flow 交互）。
 *
 * 布局：卡片横向流式排列，中央卡片正对相机，两侧卡片后退并向内偏转，
 * 形成画廊纵深；地面渐变网格增强空间感。
 * 交互：滚轮/方向键步进切换、拖拽滑动（松手吸附）、点击中央卡打开、
 *       点击侧卡聚焦、中央卡信息由 onCenterChange 供 HTML 信息条展示。
 */
import * as THREE from 'three'
import { EffectComposer } from 'three/examples/jsm/postprocessing/EffectComposer.js'
import { RenderPass } from 'three/examples/jsm/postprocessing/RenderPass.js'
import { UnrealBloomPass } from 'three/examples/jsm/postprocessing/UnrealBloomPass.js'
import { OutputPass } from 'three/examples/jsm/postprocessing/OutputPass.js'
import { makeCardTexture, makeSheenTexture, repaintCard, type CardPaintOptions } from './textures'
import type { ThemeSkin } from '../stores/skin'
import type { BookmarkItem } from '../api'

/** 皮肤 -> 场景配色（背景/雾/地面）。 */
/**
 * 皮肤场景配色：bg 兼作雾色与渐变底色，top 为渐变天空顶色，
 * glow 为地板中央舞台光斑色（取皮肤 accent，营造环境反光）。
 */
const SKIN_COLORS: Record<
  ThemeSkin,
  { bg: number; top: number; glow: number; fogNear: number; fogFar: number; floor: number; ambient: number }
> = {
  dark: { bg: 0x0d1119, top: 0x182032, glow: 0x4f8cff, fogNear: 9, fogFar: 24, floor: 0x121828, ambient: 0.85 },
  light: { bg: 0xe2e8f2, top: 0xf7f9fd, glow: 0x3b6fe0, fogNear: 10, fogFar: 26, floor: 0xdce3ee, ambient: 1.15 },
  neon: { bg: 0x0c0a18, top: 0x1a1436, glow: 0xb26bff, fogNear: 9, fogFar: 24, floor: 0x110e20, ambient: 0.7 },
  chalkboard: { bg: 0x1c2922, top: 0x2e4038, glow: 0xe5c07b, fogNear: 9, fogFar: 24, floor: 0x22332b, ambient: 0.8 },
  obsidian: { bg: 0x0a0a10, top: 0x181828, glow: 0x7c6cff, fogNear: 9, fogFar: 22, floor: 0x0e0e16, ambient: 0.65 },
  bamboo: { bg: 0xe4efdb, top: 0xf6faef, glow: 0x4a8b3b, fogNear: 10, fogFar: 26, floor: 0xd9e8c6, ambient: 1.15 },
  jade: { bg: 0x0d1b18, top: 0x1a332e, glow: 0x38b894, fogNear: 9, fogFar: 24, floor: 0x122421, ambient: 0.75 },
  rainbow: { bg: 0x120f1c, top: 0x241f3a, glow: 0xff6b9d, fogNear: 9, fogFar: 24, floor: 0x181528, ambient: 0.7 },
}

export interface EngineEvents {
  /** 点击中央卡片（打开书签） */
  onBookmarkClick: (bm: BookmarkItem) => void
  /** 画廊中央聚焦的卡片变化（HTML 信息条展示） */
  onCenterChange: (bm: BookmarkItem | null) => void
}

interface CardEntry {
  bm: BookmarkItem
  group: THREE.Group
  card: THREE.Mesh
  sheen: THREE.Mesh // 流光层（加色混合的斜向光带）
}

// 画廊几何常量
const CARD_W = 3.1
const CARD_H = 1.97
const SPACING = 2.9 // 卡片横向间距
const BEND = 1.25 // 两侧后退弯曲指数
const DEPTH_STEP = 0.62 // 每格后退深度
const TILT = 0.52 // 侧卡朝向中心的偏转角（弧度）
const VISIBLE_RANGE = 7 // 两侧最多渲染卡片数
const WHEEL_STEP = 60 // 滚轮一格累计阈值

export class SpaceEngine {
  private renderer!: THREE.WebGLRenderer
  private scene!: THREE.Scene
  private camera!: THREE.PerspectiveCamera
  private raycaster = new THREE.Raycaster()
  private pointer = new THREE.Vector2()

  private entries: CardEntry[] = [] // 当前画廊顺序
  private byId = new Map<number, BookmarkItem>()
  private cardPaints = new Map<number, CardPaintOptions>() // 建卡参数，favicon 到位后整卡重绘
  private cardCanvases = new Map<number, HTMLCanvasElement>() // 正面纹理画布引用
  private cardMeshes: THREE.Mesh[] = [] // 参与射线检测的卡片正面
  private faviconDone = new Set<number>()
  private themeColor = '#4f8cff'
  private sheenTex!: THREE.CanvasTexture
  private maxAniso = 4
  private composer!: EffectComposer
  private bloomPass!: UnrealBloomPass

  // 中央卡鼠标按压 tilt（rotate + 下沉）
  private tiltTarget = { rx: 0, ry: 0, y: 0, z: 0 }
  private tiltCur = { rx: 0, ry: 0, y: 0, z: 0 }

  private index = 0 // 浮点画廊位置
  private indexTarget = 0
  private lastCenterId: number | null = null
  private wheelAcc = 0
  private dragging = false
  private dragStartX = 0
  private dragStartIndex = 0
  private hoverMesh: THREE.Mesh | null = null
  /** 点击翻转动画：entryIndex + 起始时间（秒） */
  private flip: { entryIndex: number; start: number } | null = null
  private flipDuration = 0.65

  private floor!: THREE.Mesh
  private floorGlow!: THREE.Mesh
  private sky!: THREE.Mesh
  private skin: ThemeSkin = 'dark'
  private clock = new THREE.Clock()
  private disposed = false
  private raf = 0

  private events: EngineEvents
  private container: HTMLElement

  constructor(container: HTMLElement, events: EngineEvents) {
    this.container = container
    this.events = events
    this.init()
  }

  // ---------- 初始化 ----------

  private init() {
    const w = this.container.clientWidth
    const h = this.container.clientHeight

    this.renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false })
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    this.renderer.setSize(w, h)
    this.maxAniso = this.renderer.capabilities.getMaxAnisotropy()
    this.container.appendChild(this.renderer.domElement)

    this.scene = new THREE.Scene()
    const c = SKIN_COLORS[this.skin]
    // 背景交给渐变天空球（buildSky），雾色与渐变底色衔接
    this.scene.fog = new THREE.Fog(c.bg, c.fogNear, c.fogFar)

    this.camera = new THREE.PerspectiveCamera(50, w / h, 0.1, 100)
    this.camera.position.set(0, 0.35, 6.4)
    this.camera.lookAt(0, 0, 0)

    this.scene.add(new THREE.AmbientLight(0xffffff, c.ambient))
    const key = new THREE.DirectionalLight(0xffffff, 0.6)
    key.position.set(2, 6, 5)
    this.scene.add(key)
    const fill = new THREE.DirectionalLight(0x8899ff, 0.2)
    fill.position.set(-5, 3, -4)
    this.scene.add(fill)

    // 后处理：UnrealBloom 辉光（弱化：仅保留柔光不干扰阅读）
    // EffectComposer 默认 render target 无 MSAA，会导致整条后处理链丢失抗锯齿；
    // 显式使用 4x 多重采样 + HalfFloat，保证卡片边缘清晰。
    const renderTarget = new THREE.WebGLRenderTarget(w, h, {
      samples: 4,
      type: THREE.HalfFloatType,
    })
    this.composer = new EffectComposer(this.renderer, renderTarget)
    this.composer.addPass(new RenderPass(this.scene, this.camera))
    this.bloomPass = new UnrealBloomPass(new THREE.Vector2(w, h), 0.16, 0.4, 0.72)
    this.composer.addPass(this.bloomPass)
    this.composer.addPass(new OutputPass())

    // 共享流光纹理（offset 全局同步扫过）
    this.sheenTex = makeSheenTexture()

    // 地面：画廊反光地板（卡片悬浮其上）
    const geo = new THREE.PlaneGeometry(60, 40)
    geo.rotateX(-Math.PI / 2)
    const mat = new THREE.MeshStandardMaterial({
      color: c.floor,
      roughness: 0.35,
      metalness: 0.5,
    })
    this.floor = new THREE.Mesh(geo, mat)
    this.floor.position.y = -CARD_H / 2 - 0.85
    this.scene.add(this.floor)

    this.buildSky()
    this.buildFloorGlow()

    this.bindEvents()
    this.animate()
  }

  // ---------- 数据 ----------

  /** 设置画廊卡片（已按顺序排列的书签列表）与主题色。 */
  setCards(cards: BookmarkItem[], themeColor: string) {
    this.clearCards()
    this.themeColor = themeColor
    for (const bm of cards) {
      this.byId.set(bm.id, bm)
      const { group, card, sheen } = this.makeCard(bm)
      this.scene.add(group)
      this.entries.push({ bm, group, card, sheen })
      this.cardMeshes.push(card)
    }
    this.index = 0
    this.indexTarget = 0
    this.lastCenterId = null
    this.emitCenter()
  }

  private makeCard(bm: BookmarkItem): { group: THREE.Group; card: THREE.Mesh; sheen: THREE.Mesh } {
    const group = new THREE.Group()
    const opts: CardPaintOptions = {
      title: bm.title,
      host: safeHost(bm.url),
      color: this.themeColor,
      pinned: bm.pinned,
      faviconImg: null,
    }
    this.cardPaints.set(bm.id, opts)

    // 玻璃卡：无实心画框；正面 canvas 圆角外透明 + 主题色渐变光边，
    // 侧面/背面半透明，透出场景背景与光斑。
    const front = makeCardTexture(opts)
    const side = new THREE.MeshStandardMaterial({
      color: 0x2a3247,
      roughness: 0.7,
      metalness: 0.2,
      transparent: true,
      opacity: 0.28,
    })
    const backMat = new THREE.MeshBasicMaterial({
      color: new THREE.Color(this.themeColor),
      transparent: true,
      opacity: 0.3,
      depthWrite: false,
    })
    const card = new THREE.Mesh(
      new THREE.BoxGeometry(CARD_W, CARD_H, 0.12),
      [side, side.clone(), side.clone(), side.clone(),
        new THREE.MeshBasicMaterial({ map: front, transparent: true }),
        backMat],
    )
    card.position.z = 0.02
    card.userData = { kind: 'card', bmId: bm.id }
    group.add(card)

    // 流光层：加色混合的斜向光带，叠加在卡片正面上方
    const sheen = new THREE.Mesh(
      new THREE.PlaneGeometry(CARD_W, CARD_H),
      new THREE.MeshBasicMaterial({
        map: this.sheenTex,
        transparent: true,
        opacity: 0.16,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
      }),
    )
    sheen.position.z = 0.1
    sheen.renderOrder = 1
    group.add(sheen)

    // 登记正面纹理画布，供 favicon 到位后整卡重绘
    const frontMat = (card.material as THREE.Material[])[4] as THREE.MeshBasicMaterial
    if (frontMat.map?.image instanceof HTMLCanvasElement) {
      this.cardCanvases.set(bm.id, frontMat.map.image)
    }
    // 斜视角纹理清晰度：各向异性过滤拉满
    if (frontMat.map) frontMat.map.anisotropy = this.maxAniso
    return { group, card, sheen }
  }

  private clearCards() {
    for (const e of this.entries) {
      this.scene.remove(e.group)
      e.group.traverse((obj) => {
        const m = obj as THREE.Mesh
        if (m.geometry) m.geometry.dispose()
        const mat = m.material as THREE.Material | THREE.Material[]
        if (Array.isArray(mat)) mat.forEach((x) => x.dispose())
        else mat?.dispose()
      })
    }
    this.entries = []
    this.cardMeshes = []
    this.byId.clear()
    this.cardPaints.clear()
    this.cardCanvases.clear()
    this.faviconDone.clear()
    this.hoverMesh = null
    this.tiltTarget = { rx: 0, ry: 0, y: 0, z: 0 }
    this.tiltCur = { rx: 0, ry: 0, y: 0, z: 0 }
    this.index = 0
    this.indexTarget = 0
    this.lastCenterId = null
  }

  /** favicon 加载完成后整卡重绘（模糊背景 + 玻璃 + favicon 徽标）。 */
  setCardFavicon(bmId: number, src: string) {
    if (this.faviconDone.has(bmId)) return
    const card = this.entries.find((en) => en.bm.id === bmId)?.card
    const opts = this.cardPaints.get(bmId)
    const canvas = this.cardCanvases.get(bmId)
    if (!card || !opts || !canvas) return
    const frontMat = (card.material as THREE.Material[])[4] as THREE.MeshBasicMaterial
    if (!frontMat.map) return
    this.faviconDone.add(bmId)
    const img = new Image()
    img.onload = () => {
      repaintCard(canvas, frontMat.map!, { ...opts, faviconImg: img })
    }
    img.src = src
  }

  // ---------- 交互 ----------

  private bindEvents() {
    const el = this.renderer.domElement
    el.addEventListener('pointermove', this.onPointerMove)
    el.addEventListener('pointerdown', this.onPointerDown)
    el.addEventListener('pointerup', this.onPointerUp)
    el.addEventListener('wheel', this.onWheel, { passive: false })
    window.addEventListener('keydown', this.onKey)
    window.addEventListener('resize', this.onResize)
  }

  private onPointerMove = (e: PointerEvent) => {
    const rect = this.renderer.domElement.getBoundingClientRect()
    this.pointer.x = ((e.clientX - rect.left) / rect.width) * 2 - 1
    this.pointer.y = -((e.clientY - rect.top) / rect.height) * 2 + 1

    if (this.dragging) {
      const dx = e.clientX - this.dragStartX
      this.indexTarget = clampF(this.dragStartIndex + dx / 90, 0, Math.max(this.entries.length - 1, 0))
      return
    }

    // hover 光标 + 中央卡按压 tilt 目标
    this.raycaster.setFromCamera(this.pointer, this.camera)
    const hits = this.raycaster.intersectObjects(this.cardMeshes, false)
    const hit = hits[0]
    this.hoverMesh = (hit?.object as THREE.Mesh) ?? null
    this.renderer.domElement.style.cursor = this.hoverMesh ? 'pointer' : 'default'

    const centerIdx = Math.round(this.index)
    const centerEntry = this.entries[centerIdx]
    if (hit && centerEntry && hit.object === centerEntry.card) {
      // 命中点 -> 卡片局部坐标 -> 归一化 (-1..1)
      const local = centerEntry.card.worldToLocal(hit.point.clone())
      const nx = clampF(local.x / (CARD_W / 2), -1, 1)
      const ny = clampF(local.y / (CARD_H / 2), -1, 1)
      // 按压：向鼠标方向倾斜 + 下沉靠近
      this.tiltTarget.rx = ny * 0.16
      this.tiltTarget.ry = -nx * 0.2
      this.tiltTarget.y = -0.05
      this.tiltTarget.z = 0.06
    } else {
      this.tiltTarget = { rx: 0, ry: 0, y: 0, z: 0 }
    }
  }

  private onPointerDown = (e: PointerEvent) => {
    if (e.button !== 0) return
    this.dragging = true
    this.dragStartX = e.clientX
    this.dragStartIndex = this.indexTarget
  }

  private onPointerUp = (e: PointerEvent) => {
    if (e.button !== 0) return
    const moved = Math.abs(e.clientX - this.dragStartX)
    this.dragging = false
    this.indexTarget = clampF(Math.round(this.indexTarget), 0, Math.max(this.entries.length - 1, 0))
    if (moved <= 6) {
      // 点击：命中卡在中央 -> 打开；否则聚焦到该卡
      this.pointer.x = ((e.clientX - this.renderer.domElement.getBoundingClientRect().left) / this.renderer.domElement.clientWidth) * 2 - 1
      this.pointer.y = -((e.clientY - this.renderer.domElement.getBoundingClientRect().top) / this.renderer.domElement.clientHeight) * 2 + 1
      this.raycaster.setFromCamera(this.pointer, this.camera)
      const hits = this.raycaster.intersectObjects(this.cardMeshes, false)
      const hit = hits[0]?.object as THREE.Mesh | undefined
      if (hit) {
        const bmId = (hit.userData as { bmId: number }).bmId
        const idx = this.entries.findIndex((en) => en.bm.id === bmId)
        if (idx >= 0) {
          if (Math.abs(idx - this.indexTarget) < 0.5) {
            // 点击中央卡：先翻转展示背面，复位后再打开
            this.startFlip(idx)
          } else {
            this.indexTarget = idx
          }
        }
      }
    }
  }

  private onWheel = (e: WheelEvent) => {
    e.preventDefault()
    this.wheelAcc += e.deltaY
    while (Math.abs(this.wheelAcc) >= WHEEL_STEP) {
      const dir = this.wheelAcc > 0 ? 1 : -1
      this.wheelAcc -= dir * WHEEL_STEP
      this.indexTarget = clampF(Math.round(this.indexTarget) + dir, 0, Math.max(this.entries.length - 1, 0))
    }
  }

  private onKey = (e: KeyboardEvent) => {
    if (e.key === 'ArrowRight') {
      this.indexTarget = clampF(Math.round(this.indexTarget) + 1, 0, Math.max(this.entries.length - 1, 0))
    } else if (e.key === 'ArrowLeft') {
      this.indexTarget = clampF(Math.round(this.indexTarget) - 1, 0, Math.max(this.entries.length - 1, 0))
    } else if (e.key === 'Enter') {
      this.startFlip(Math.round(this.index))
    }
  }

  /** 启动中央卡翻转动画；结束后回调打开书签。翻转期间忽略重复触发。 */
  private startFlip(entryIndex: number) {
    if (this.flip || !this.entries[entryIndex]) return
    this.flip = { entryIndex, start: this.clock.getElapsedTime() }
  }

  private onResize = () => {
    if (this.disposed) return
    const w = this.container.clientWidth
    const h = this.container.clientHeight
    if (!w || !h) return
    this.camera.aspect = w / h
    this.camera.updateProjectionMatrix()
    this.renderer.setSize(w, h)
    this.composer.setSize(w, h)
    this.bloomPass.resolution.set(w, h)
  }

  /** 复位到第一张卡。 */
  resetView() {
    this.indexTarget = 0
  }

  /** 渐变天空球：BackSide 球体 + 垂直渐变 shader，替代纯色背景。 */
  private buildSky() {
    const c = SKIN_COLORS[this.skin]
    const mat = new THREE.ShaderMaterial({
      uniforms: {
        topColor: { value: new THREE.Color(c.top) },
        bottomColor: { value: new THREE.Color(c.bg) },
        offset: { value: 6 },
        exponent: { value: 0.85 },
      },
      vertexShader: `
        varying vec3 vWorldPosition;
        void main() {
          vec4 worldPosition = modelMatrix * vec4(position, 1.0);
          vWorldPosition = worldPosition.xyz;
          gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
        }`,
      fragmentShader: `
        uniform vec3 topColor;
        uniform vec3 bottomColor;
        uniform float offset;
        uniform float exponent;
        varying vec3 vWorldPosition;
        void main() {
          float h = normalize(vWorldPosition + vec3(0.0, offset, 0.0)).y;
          gl_FragColor = vec4(mix(bottomColor, topColor, max(pow(max(h, 0.0), exponent), 0.0)), 1.0);
        }`,
      side: THREE.BackSide,
      depthWrite: false,
    })
    this.sky = new THREE.Mesh(new THREE.SphereGeometry(36, 24, 16), mat)
    this.scene.add(this.sky)
  }

  /** 地板舞台光斑：accent 色径向渐变，additive 弱透明，营造环境反光。 */
  private buildFloorGlow() {
    const c = SKIN_COLORS[this.skin]
    const S = 256
    const canvas = document.createElement('canvas')
    canvas.width = S
    canvas.height = S
    const ctx = canvas.getContext('2d')!
    const grad = ctx.createRadialGradient(S / 2, S / 2, 8, S / 2, S / 2, S / 2)
    grad.addColorStop(0, 'rgba(255,255,255,0.85)')
    grad.addColorStop(0.5, 'rgba(255,255,255,0.25)')
    grad.addColorStop(1, 'rgba(255,255,255,0)')
    ctx.fillStyle = grad
    ctx.fillRect(0, 0, S, S)
    const tex = new THREE.CanvasTexture(canvas)
    tex.colorSpace = THREE.SRGBColorSpace

    const mat = new THREE.MeshBasicMaterial({
      color: c.glow,
      map: tex,
      transparent: true,
      opacity: 0.15,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    })
    const geo = new THREE.PlaneGeometry(16, 9)
    geo.rotateX(-Math.PI / 2)
    this.floorGlow = new THREE.Mesh(geo, mat)
    this.floorGlow.position.y = this.floor.position.y + 0.03
    this.scene.add(this.floorGlow)
  }

  // ---------- 皮肤 ----------

  applySkin(skin: ThemeSkin) {
    this.skin = skin
    const c = SKIN_COLORS[skin]
    if (this.scene.fog) {
      ;(this.scene.fog as THREE.Fog).color.setHex(c.bg)
      ;(this.scene.fog as THREE.Fog).near = c.fogNear
      ;(this.scene.fog as THREE.Fog).far = c.fogFar
    }
    ;(this.floor.material as THREE.MeshStandardMaterial).color.setHex(c.floor)
    const skyMat = this.sky.material as THREE.ShaderMaterial
    ;(skyMat.uniforms.topColor.value as THREE.Color).setHex(c.top)
    ;(skyMat.uniforms.bottomColor.value as THREE.Color).setHex(c.bg)
    ;(this.floorGlow.material as THREE.MeshBasicMaterial).color.setHex(c.glow)
    const amb = this.scene.children.find((o) => o instanceof THREE.AmbientLight) as THREE.AmbientLight
    if (amb) amb.intensity = c.ambient
  }

  // ---------- 渲染循环 ----------

  /** 按当前浮点位置布置每张卡片：中央正对、两侧后退偏转。 */
  private layoutCards() {
    for (let i = 0; i < this.entries.length; i++) {
      const d = i - this.index
      const group = this.entries[i].group
      const ad = Math.abs(d)
      group.visible = ad < VISIBLE_RANGE
      if (!group.visible) continue
      group.position.x = d * SPACING
      group.position.y = Math.sin(ad * 0.8) * -0.06
      group.position.z = -Math.pow(ad, BEND) * DEPTH_STEP
      group.rotation.y = -clampF(d, -1, 1) * TILT
      // 中央卡轻微上浮强调
      if (ad < 0.5) group.position.y += 0.1
    }
  }

  private emitCenter() {
    const i = Math.round(this.index)
    const bm = this.entries[i]?.bm ?? null
    const id = bm?.id ?? null
    if (id !== this.lastCenterId) {
      this.lastCenterId = id
      this.events.onCenterChange(bm)
    }
  }

  private animate = () => {
    if (this.disposed) return
    this.raf = requestAnimationFrame(this.animate)
    const t = this.clock.getElapsedTime()

    this.index += (this.indexTarget - this.index) * 0.14
    // 松手/滚轮停止后吸附最近卡
    if (!this.dragging && Math.abs(this.indexTarget - this.index) < 0.005) {
      this.indexTarget = clampF(Math.round(this.index), 0, Math.max(this.entries.length - 1, 0))
    }
    this.layoutCards()

    // 中央卡呼吸强调 + hover 微升 + 按压 tilt
    this.tiltCur.rx += (this.tiltTarget.rx - this.tiltCur.rx) * 0.12
    this.tiltCur.ry += (this.tiltTarget.ry - this.tiltCur.ry) * 0.12
    this.tiltCur.y += (this.tiltTarget.y - this.tiltCur.y) * 0.12
    this.tiltCur.z += (this.tiltTarget.z - this.tiltCur.z) * 0.12
    for (let i = 0; i < this.entries.length; i++) {
      const en = this.entries[i]
      if (!en.group.visible) continue
      const d = Math.abs(i - this.index)
      if (d < 0.5) {
        const targetY = 0.08 + (en.card === this.hoverMesh ? 0.04 : 0) + Math.sin(t * 1.6) * 0.015 + this.tiltCur.y
        en.group.position.y += (targetY - en.group.position.y) * 0.15
        en.group.position.z += this.tiltCur.z
        en.group.rotation.x = this.tiltCur.rx
        const flipping = this.flip && this.flip.entryIndex === i
        if (!flipping) en.group.rotation.y += this.tiltCur.ry
        // 流光：进一步弱化，仅余隐约流动感
        ;(en.sheen.material as THREE.MeshBasicMaterial).opacity = 0.06 + Math.sin(t * 1.4) * 0.02
      } else {
        ;(en.sheen.material as THREE.MeshBasicMaterial).opacity = 0.035
      }
    }

    // 流光扫动（共享纹理全局平移，慢速）
    this.sheenTex.offset.x = (t * 0.1) % 1

    // 点击翻转：360° 缓动旋转，结束后回调打开
    if (this.flip) {
      const p = Math.min((t - this.flip.start) / this.flipDuration, 1)
      const en = this.entries[this.flip.entryIndex]
      if (!en) {
        this.flip = null
      } else {
        const eased = p < 0.5 ? 4 * p * p * p : 1 - Math.pow(-2 * p + 2, 3) / 2
        en.group.rotation.y = eased * Math.PI * 2
        if (p >= 1) {
          const bm = en.bm
          this.flip = null
          this.events.onBookmarkClick(bm)
        }
      }
    }

    this.emitCenter()
    this.composer.render()
  }

  dispose() {
    this.disposed = true
    cancelAnimationFrame(this.raf)
    const el = this.renderer.domElement
    el.removeEventListener('pointermove', this.onPointerMove)
    el.removeEventListener('pointerdown', this.onPointerDown)
    el.removeEventListener('pointerup', this.onPointerUp)
    el.removeEventListener('wheel', this.onWheel)
    window.removeEventListener('keydown', this.onKey)
    window.removeEventListener('resize', this.onResize)
    this.clearCards()
    this.floor.geometry.dispose()
    ;(this.floor.material as THREE.Material).dispose()
    this.floorGlow.geometry.dispose()
    ;(this.floorGlow.material as THREE.Material).dispose()
    this.sky.geometry.dispose()
    ;(this.sky.material as THREE.Material).dispose()
    this.sheenTex.dispose()
    this.bloomPass.dispose()
    this.composer.dispose()
    this.renderer.dispose()
    el.remove()
  }
}

function safeHost(url: string): string {
  try {
    return new URL(url.includes('://') ? url : `https://${url}`).host
  } catch {
    return url.slice(0, 30)
  }
}

function clampF(v: number, lo: number, hi: number): number {
  if (hi < lo) return lo
  if (v < lo) return lo
  if (v > hi) return hi
  return v
}
