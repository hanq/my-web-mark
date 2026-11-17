/**
 * 2.5D 书签空间场景引擎（Three.js）。
 *
 * 布局：正交相机 45° 等轴视角；每个主题一个圆形岛台，书签卡片按网格
 * 排列在台面上；岛台间环绕排布于中央广场周围。
 * 交互：滚轮缩放 / 左键拖拽平移 / 点击书签打开 / 点击岛台聚焦 /
 *       hover 高亮卡片；Esc 或点空白复位视角。
 */
import * as THREE from 'three'
import { makeCardTexture, makeCardBackTexture, makeLabelTexture } from './textures'
import type { ThemeSkin } from '../stores/skin'
import type { BookmarkItem, ThemeSummary } from '../api'

/** 皮肤 -> 场景配色映射。 */
const SKIN_COLORS: Record<ThemeSkin, { bg: number; fog: number; grid: number; island: number; ambient: number }> = {
  dark: { bg: 0x0f121c, fog: 0x0f121c, grid: 0x26304a, island: 0x1e2740, ambient: 0.85 },
  light: { bg: 0xeef2f9, fog: 0xeef2f9, grid: 0xc9d3e6, island: 0xffffff, ambient: 1.1 },
  neon: { bg: 0x0a0a14, fog: 0x0a0a14, grid: 0x35356e, island: 0x1b1b42, ambient: 0.7 },
}

export interface EngineEvents {
  onBookmarkClick: (bm: BookmarkItem) => void
  onThemeClick: (themeId: number) => void
  onHover: (text: string | null, x: number, y: number) => void
}

interface IslandData {
  themeId: number
  group: THREE.Group
  label: THREE.Sprite
  cards: BookmarkItem[]
}

const CARD_W = 3.1
const CARD_H = 1.97
const GRID_X = 3.6
const GRID_Z = 2.6
const ISLAND_R = 7.2 // 岛台半径（容纳 3x3 卡片）

export class SpaceEngine {
  private renderer!: THREE.WebGLRenderer
  private scene!: THREE.Scene
  private camera!: THREE.OrthographicCamera
  private raycaster = new THREE.Raycaster()
  private pointer = new THREE.Vector2()
  private clock = new THREE.Clock()

  private islands: IslandData[] = []
  private cardMeshes = new Map<string, THREE.Mesh>() // "bmId" -> mesh
  private grid!: THREE.GridHelper

  private hovered: THREE.Mesh | null = null
  private focusTarget: THREE.Vector3 | null = null
  private focusLookAt: THREE.Vector3 | null = null
  private camBase = new THREE.Vector3(26, 26, 26)
  private camLook = new THREE.Vector3(0, 0, 0)
  private panning = false
  private lastX = 0
  private lastY = 0
  private zoom = 1
  private disposed = false
  private raf = 0

  private skin: ThemeSkin = 'dark'
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
    this.container.appendChild(this.renderer.domElement)

    this.scene = new THREE.Scene()
    this.scene.background = new THREE.Color(SKIN_COLORS[this.skin].bg)
    this.scene.fog = new THREE.Fog(SKIN_COLORS[this.skin].fog, 55, 110)

    const aspect = w / h
    const viewSize = 20
    this.camera = new THREE.OrthographicCamera(
      (-viewSize * aspect) / 2, (viewSize * aspect) / 2,
      viewSize / 2, -viewSize / 2,
      0.1, 300,
    )
    this.camBase.set(26, 30, 26)
    this.camera.position.copy(this.camBase)
    this.camera.lookAt(this.camLook)

    // 灯光：环境光 + 平行光，柔和立体
    const ambient = new THREE.AmbientLight(0xffffff, SKIN_COLORS[this.skin].ambient)
    this.scene.add(ambient)
    const dir = new THREE.DirectionalLight(0xffffff, 0.9)
    dir.position.set(18, 30, 12)
    this.scene.add(dir)
    const fill = new THREE.DirectionalLight(0x8899ff, 0.25)
    fill.position.set(-14, 18, -16)
    this.scene.add(fill)

    // 地面网格
    this.grid = new THREE.GridHelper(90, 30, SKIN_COLORS[this.skin].grid, SKIN_COLORS[this.skin].grid)
    ;(this.grid.material as THREE.Material).transparent = true
    ;(this.grid.material as THREE.Material).opacity = 0.35
    this.grid.position.y = -0.6
    this.scene.add(this.grid)

    this.bindEvents()
    this.animate()
  }

  // ---------- 数据构建 ----------

  /** 用主题+书签数据重建全部岛台（数据变化时调用）。 */
  build(themes: ThemeSummary[], bookmarks: BookmarkItem[]) {
    this.clearIslands()

    const withBm = themes.filter((t) => bookmarks.some((b) => b.themeId === t.id))
    const emptyThemes = themes.filter((t) => !bookmarks.some((b) => b.themeId === t.id))
    // 空主题不显示岛台（保持场景整洁）
    const list = [...withBm]
    if (!list.length && !emptyThemes.length) return
    if (!list.length) return

    // 岛台环绕布局：中央广场 + 圆周排布
    const n = list.length
    const ringRadius = n === 1 ? 0 : Math.max(13, n * 4.2)
    list.forEach((theme, i) => {
      const angle = (i / n) * Math.PI * 2 - Math.PI / 2
      const x = n === 1 ? 0 : Math.cos(angle) * ringRadius
      const z = n === 1 ? 0 : Math.sin(angle) * ringRadius * 0.78 // 椭圆排布更符合宽屏
      this.buildIsland(theme, bookmarks.filter((b) => b.themeId === theme.id), x, z)
    })
  }

  private buildIsland(theme: ThemeSummary, cards: BookmarkItem[], x: number, z: number) {
    const group = new THREE.Group()
    group.position.set(x, 0, z)
    const islandColor = SKIN_COLORS[this.skin].island

    // 岛台：扁圆柱
    const cylGeo = new THREE.CylinderGeometry(ISLAND_R, ISLAND_R * 1.02, 1.1, 48)
    const cylMat = new THREE.MeshStandardMaterial({
      color: islandColor,
      roughness: 0.85,
      metalness: 0.1,
    })
    const cyl = new THREE.Mesh(cylGeo, cylMat)
    cyl.position.y = -0.05
    group.add(cyl)

    // 主题色发光环
    const ringGeo = new THREE.TorusGeometry(ISLAND_R + 0.18, 0.09, 12, 64)
    const ringMat = new THREE.MeshBasicMaterial({ color: new THREE.Color(theme.color) })
    const ring = new THREE.Mesh(ringGeo, ringMat)
    ring.rotation.x = Math.PI / 2
    ring.position.y = 0.5
    group.add(ring)

    // 名牌 sprite（岛台上方）
    const labelMat = new THREE.SpriteMaterial({
      map: makeLabelTexture(theme.name, theme.color, theme.bookmarkCount),
      transparent: true,
    })
    const label = new THREE.Sprite(labelMat)
    label.scale.set(7.2, 1.8, 1)
    label.position.y = 4.2
    group.add(label)

    // 书签卡片网格：最多 9 个/岛（3x3），超出部分显示 "+N"
    const show = cards.slice(0, 9)
    const cols = Math.min(3, show.length)
    show.forEach((bm, i) => {
      const col = i % cols
      const row = Math.floor(i / cols)
      const cx = (col - (cols - 1) / 2) * GRID_X
      const cz = (row - (Math.ceil(show.length / cols) - 1) / 2) * GRID_Z
      const mesh = this.makeCard(bm, theme.color)
      mesh.position.set(cx, 0.62, cz)
      // 轻微朝向中心，形成围坐感
      mesh.lookAt(new THREE.Vector3(x, 0.62, z))
      group.add(mesh)
      this.cardMeshes.set(String(bm.id), mesh)
    })

    // 溢出提示
    if (cards.length > 9) {
      const more = new THREE.Sprite(new THREE.SpriteMaterial({
        map: makeLabelTexture(`+${cards.length - 9} 更多`, theme.color, 0),
        transparent: true,
      }))
      more.scale.set(4.6, 1.15, 1)
      more.position.set(0, 0.9, ISLAND_R - 1.6)
      group.add(more)
    }

    this.scene.add(group)
    this.islands.push({ themeId: theme.id, group, label, cards })
  }

  private makeCard(bm: BookmarkItem, themeColor: string): THREE.Mesh {
    const host = safeHost(bm.url)
    const front = makeCardTexture({
      title: bm.title,
      host,
      color: themeColor,
      textColor: '#1c2333',
      pinned: bm.pinned,
    })
    const back = makeCardBackTexture(themeColor)
    const mats = [
      sideMat(), sideMat(), topMat(), topMat(),
      new THREE.MeshBasicMaterial({ map: front }), // +z 面朝相机方向
      new THREE.MeshBasicMaterial({ map: back }),
    ]
    const geo = new THREE.BoxGeometry(CARD_W, 0.16, CARD_H)
    const mesh = new THREE.Mesh(geo, mats)
    mesh.userData = { kind: 'bookmark', bm }
    return mesh
  }

  private clearIslands() {
    for (const isl of this.islands) {
      this.scene.remove(isl.group)
      isl.group.traverse((obj) => {
        const m = obj as THREE.Mesh
        if (m.geometry) m.geometry.dispose()
        const mat = m.material as THREE.Material | THREE.Material[]
        if (Array.isArray(mat)) mat.forEach((x) => x.dispose())
        else mat?.dispose()
      })
    }
    this.islands = []
    this.cardMeshes.clear()
    this.hovered = null
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

    if (this.panning) {
      const dx = e.clientX - this.lastX
      const dy = e.clientY - this.lastY
      this.lastX = e.clientX
      this.lastY = e.clientY
      // 平移方向随相机朝向（等轴视角下屏幕 x/y 对应世界 x/-z）
      const k = 0.045 * this.zoom
      this.camLook.x -= dx * k
      this.camLook.z -= dy * k
      this.focusTarget = null
      this.focusLookAt = null
      return
    }

    // hover 检测
    this.raycaster.setFromCamera(this.pointer, this.camera)
    const meshes = [...this.cardMeshes.values()]
    const hits = this.raycaster.intersectObjects(meshes, false)
    const first = hits.find((h) => h.object instanceof THREE.Mesh)
    const mesh = (first?.object as THREE.Mesh) ?? null
    if (mesh !== this.hovered) {
      this.hovered = mesh
      this.renderer.domElement.style.cursor = mesh ? 'pointer' : 'default'
    }
    if (mesh) {
      const bm = (mesh.userData as { bm: BookmarkItem }).bm
      this.events.onHover(`${bm.title} — ${safeHost(bm.url)}`, e.clientX, e.clientY)
    } else {
      this.events.onHover(null, 0, 0)
    }
  }

  private onPointerDown = (e: PointerEvent) => {
    if (e.button !== 0) return
    this.panning = true
    this.lastX = e.clientX
    this.lastY = e.clientY
  }

  private onPointerUp = (e: PointerEvent) => {
    if (e.button !== 0) return
    const moved = Math.abs(e.clientX - this.lastX) + Math.abs(e.clientY - this.lastY)
    this.panning = false
    if (moved > 6) return // 拖拽不算点击

    this.raycaster.setFromCamera(this.pointer, this.camera)
    const bmHits = this.raycaster.intersectObjects([...this.cardMeshes.values()], false)
    if (bmHits.length) {
      const bm = (bmHits[0].object.userData as { bm: BookmarkItem }).bm
      this.events.onBookmarkClick(bm)
      return
    }
    // 点岛台 -> 聚焦主题
    const islandHits = this.raycaster.intersectObjects(
      this.islands.map((i) => i.group), true,
    )
    if (islandHits.length) {
      let obj: THREE.Object3D | null = islandHits[0].object
      while (obj) {
        const found = this.islands.find((i) => i.group === obj)
        if (found) {
          this.events.onThemeClick(found.themeId)
          this.focusIsland(found)
          return
        }
        obj = obj.parent
      }
    }
  }

  private onWheel = (e: WheelEvent) => {
    e.preventDefault()
    const delta = Math.sign(e.deltaY)
    this.zoom = THREE.MathUtils.clamp(this.zoom + delta * 0.1, 0.45, 2.4)
    this.focusTarget = null
    this.focusLookAt = null
  }

  private onKey = (e: KeyboardEvent) => {
    if (e.key === 'Escape') this.resetView()
  }

  private onResize = () => {
    if (this.disposed) return
    const w = this.container.clientWidth
    const h = this.container.clientHeight
    if (!w || !h) return
    this.camera.left = (-20 * (w / h)) / 2
    this.camera.right = (20 * (w / h)) / 2
    this.camera.updateProjectionMatrix()
    this.renderer.setSize(w, h)
  }

  /** 聚焦某主题岛台（相机平移至其上方）。 */
  focusIsland(isl: IslandData) {
    this.focusLookAt = isl.group.position.clone()
    this.focusTarget = isl.group.position.clone().add(new THREE.Vector3(10, 12, 10))
  }

  /** 复位视角。 */
  resetView() {
    this.focusTarget = null
    this.focusLookAt = null
    this.camLook.set(0, 0, 0)
    this.zoom = 1
  }

  // ---------- 皮肤联动 ----------

  applySkin(skin: ThemeSkin) {
    this.skin = skin
    const c = SKIN_COLORS[skin]
    ;(this.scene.background as THREE.Color).setHex(c.bg)
    if (this.scene.fog) (this.scene.fog as THREE.Fog).color.setHex(c.fog)
    const gridM = this.grid.material as THREE.LineBasicMaterial
    gridM.color.setHex(c.grid)
    for (const isl of this.islands) {
      isl.group.traverse((obj) => {
        const mesh = obj as THREE.Mesh
        if (mesh.geometry instanceof THREE.CylinderGeometry && mesh.material) {
          (mesh.material as THREE.MeshStandardMaterial).color.setHex(c.island)
        }
      })
    }
    // 环境光
    const amb = this.scene.children.find((o) => o instanceof THREE.AmbientLight) as THREE.AmbientLight
    if (amb) amb.intensity = c.ambient
  }

  // ---------- 渲染循环 ----------

  private animate = () => {
    if (this.disposed) return
    this.raf = requestAnimationFrame(this.animate)
    const t = this.clock.getElapsedTime()

    // 相机目标位（focus / 手动 / 复位）
    const targetLook = this.focusLookAt ?? this.camLook
    const targetPos = this.focusTarget
      ? this.focusLookAt!.clone().add(new THREE.Vector3(10, 12, 10).multiplyScalar(this.zoom))
      : this.camLook.clone().add(new THREE.Vector3(26, 30, 26).multiplyScalar(this.zoom * 0.62))

    this.camera.position.lerp(targetPos, 0.08)
    this.camLook.lerp(targetLook, 0.08)
    this.camera.lookAt(this.camLook)

    // hover 卡片浮起
    for (const [, mesh] of this.cardMeshes) {
      const baseY = 0.62
      const isHover = mesh === this.hovered
      const targetY = isHover ? baseY + 0.35 : baseY + Math.sin(t * 1.6 + mesh.position.x * 2) * 0.03
      mesh.position.y += (targetY - mesh.position.y) * 0.18
    }

    // 名牌轻微浮动
    for (const isl of this.islands) {
      isl.label.position.y = 4.2 + Math.sin(t * 1.2 + isl.themeId) * 0.12
    }

    this.renderer.render(this.scene, this.camera)
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
    this.clearIslands()
    this.grid.geometry.dispose()
    ;(this.grid.material as THREE.Material).dispose()
    this.renderer.dispose()
    el.remove()
  }
}

function sideMat() {
  return new THREE.MeshStandardMaterial({ color: 0x2a3247, roughness: 0.9 })
}
function topMat() {
  return new THREE.MeshStandardMaterial({ color: 0x323c58, roughness: 0.9 })
}

function safeHost(url: string): string {
  try {
    return new URL(url.includes('://') ? url : `https://${url}`).host
  } catch {
    return url.slice(0, 30)
  }
}
