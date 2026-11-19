<script setup lang="ts">
/**
 * 线性 SVG 图标组件（iconfont/iconpark 风格：24×24、stroke 2、currentColor、圆头端点）。
 * 用法：<AppIcon name="plus" /> 或带尺寸 <AppIcon name="plus" :size="18" />
 */
const props = defineProps<{
  name: string
  size?: number
}>()

// path d 数据统一为 24×24 线性风格
const ICONS: Record<string, string[]> = {
  plus: ['M12 5v14', 'M5 12h14'],
  bookmark: ['M6 4h12v16l-6-4-6 4z'],
  lock: ['M6 11h12v9H6z', 'M8 11V8a4 4 0 0 1 8 0v3', 'M12 15v2'],
  unlock: ['M6 11h12v9H6z', 'M8 11V8a4 4 0 0 1 7.6-1.6', 'M12 15v2'],
  key: ['M15 9a4.5 4.5 0 1 0-4.5 4.5L11 13l1.5 1.5', 'M15 9l-3.2 3.2', 'M13.5 10.5 18 15l-1.5 1.5-2-2'],
  eye: ['M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6-10-6-10-6z', 'M12 9.5a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5z'],
  eyeOff: ['M4 4l16 16', 'M9.9 5.2A10 10 0 0 1 12 5c6.5 0 10 7 10 7a17 17 0 0 1-3.2 3.9', 'M6.1 7.6A16.4 16.4 0 0 0 2 12s3.5 7 10 7a10 10 0 0 0 4.2-.9', 'M9.9 9.9a2.5 2.5 0 0 0 3.4 3.5'],
  copy: ['M9 9h10v12H9z', 'M5 15H4V3h11v1'],
  trash: ['M4 7h16', 'M9 7V4h6v3', 'M6 7l1 14h10l1-14', 'M10 11v6', 'M14 11v6'],
  edit: ['M4 20h16', 'M5 16l10-10 3 3-10 10H5v-3z'],
  folder: ['M3 6h6l2 2h10v12H3z'],
  grid: ['M4 4h7v7H4z', 'M13 4h7v7h-7z', 'M4 13h7v7H4z', 'M13 13h7v7h-7z'],
  cube: ['M12 3l8 4.5v9L12 21l-8-4.5v-9L12 3z', 'M12 12l8-4.5', 'M12 12 4 7.5', 'M12 12v9'],
  list: ['M8 6h13', 'M8 12h13', 'M8 18h13', 'M4 6h.01', 'M4 12h.01', 'M4 18h.01'],
  download: ['M12 4v11', 'M7 10l5 5 5-5', 'M4 20h16'],
  refresh: ['M20 12a8 8 0 1 1-2.3-5.6', 'M20 4v5h-5'],
  close: ['M6 6l12 12', 'M18 6 6 18'],
  check: ['M5 13l4 4L19 7'],
  external: ['M14 4h6v6', 'M20 4 10 14', 'M9 5H5v14h14v-4'],
  palette: ['M12 3a9 9 0 1 0 0 18c1.5 0 2-1 2-2 0-1.5-1-2-1-3 0-1 1-2 2.5-2H18a4 4 0 0 0 4-4c0-4-4.5-7-10-7z', 'M8 9h.01', 'M6.5 13h.01', 'M11 7h.01'],
  settings: ['M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z', 'M19 12a7 7 0 0 0-.1-1.2l2-1.5-2-3.5-2.4 1a7 7 0 0 0-2-1.2L14 3h-4l-.5 2.6a7 7 0 0 0-2 1.2l-2.4-1-2 3.5 2 1.5a7 7 0 0 0 0 2.4l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 2 1.2L10 21h4l.5-2.6a7 7 0 0 0 2-1.2l2.4 1 2-3.5-2-1.5c.07-.4.1-.8.1-1.2z'],
  pin: ['M12 3l2.5 6.5L21 12l-5 4 1.5 7L12 19l-5.5 4L8 16l-5-4 6.5-2.5L12 3z'],
  star: ['M12 4l2.3 5.4 5.7.5-4.3 3.8 1.3 5.6L12 16.4 7 19.3l1.3-5.6L4 9.9l5.7-.5L12 4z'],
  chrome: ['M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z', 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z', 'M12 3v5', 'M20.5 8.5 16 11', 'M8 20l3-6'],
  shield: ['M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6l8-3z', 'M9 12l2 2 4-4'],
  monitor: ['M4 5h16v11H4z', 'M9 20h6', 'M12 16v4'],
  minimize: ['M5 12h14'],
  maximize: ['M5 5h14v14H5z'],
  restore: ['M8 8h11v11H8z', 'M16 5H5v11'],
  chevronLeft: ['M15 5l-7 7 7 7'],
  chevronRight: ['M9 5l7 7-7 7'],
  more: ['M12 5h.01', 'M12 12h.01', 'M12 19h.01'],
  globe: ['M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z', 'M3 12h18', 'M12 3c2.7 2.4 4 5.6 4 9s-1.3 6.6-4 9c-2.7-2.4-4-5.6-4-9s1.3-6.6 4-9z'],
  wifi: ['M2.5 9a14 14 0 0 1 19 0', 'M5.5 12.5a9.5 9.5 0 0 1 13 0', 'M8.5 16a5 5 0 0 1 7 0', 'M12 19.5h.01'],
  tag: ['M4 4h7l9 9-7 7-9-9V4z', 'M8 8h.01'],
  search: ['M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14z', 'M16.5 16.5 21 21'],
  link: ['M10 13a4.5 4.5 0 0 0 6.4 0l2.6-2.6a4.5 4.5 0 0 0-6.4-6.4L11 5.6', 'M14 11a4.5 4.5 0 0 0-6.4 0L5 13.6a4.5 4.5 0 0 0 6.4 6.4l1.6-1.6'],
  clipboard: ['M9 5h6v2H9z', 'M7 6H5v15h14V6h-2', 'M8 3h8v4H8z', 'M9 12h6', 'M9 16h4'],
}
</script>

<template>
  <svg
    :width="props.size ?? 20"
    :height="props.size ?? 20"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path v-for="(d, i) in ICONS[props.name] ?? []" :key="i" :d="d" />
  </svg>
</template>
