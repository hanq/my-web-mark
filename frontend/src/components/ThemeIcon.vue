<script setup lang="ts">
/** 主题图标渲染（IconPark 风格线性 SVG，从 theme_icons 注册表取 path）。 */
import { themeIconPaths } from './theme_icons'

const props = defineProps<{ name: string; size?: number }>()
</script>

<template>
  <svg
    :width="props.size ?? 18"
    :height="props.size ?? 18"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path v-for="(d, i) in themeIconPaths(props.name)" :key="i" :d="d" />
  </svg>
</template>
