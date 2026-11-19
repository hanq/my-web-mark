/**
 * 主题图标注册表：内置常用 IconPark 风格线性图标（24×24、stroke 2、
 * 圆头端点、currentColor 着色）。theme.icon 存图标名（key），
 * 渲染时查表绘制；新增图标在 THEME_ICONS 追加即可。
 */
export const THEME_ICONS: Record<string, string[]> = {
  // 开发
  code: ['M8 6 3 12l5 6', 'M16 6l5 6-5 6'],
  terminal: ['M3 5h18v14H3z', 'M7 9l3 3-3 3', 'M12 15h5'],
  // 设计 / 影音
  palette: [
    'M12 3a9 9 0 1 0 0 18c1.5 0 2-1 2-2 0-1.5-1-2-1-3 0-1 1-2 2.5-2H18a4 4 0 0 0 4-4c0-4-4.5-7-10-7z',
    'M8 9h.01', 'M6.5 13h.01', 'M11 7h.01',
  ],
  video: ['M4 5h16v14H4z', 'M10 9l5 3-5 3z'],
  music: ['M9 18V5l10-2v13', 'M9 18a2.5 2.5 0 1 1-5 0 2.5 2.5 0 0 1 5 0z', 'M19 16a2.5 2.5 0 1 1-5 0 2.5 2.5 0 0 1 5 0z'],
  camera: ['M4 8h4l2-3h4l2 3h4v12H4z', 'M12 17a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7z'],
  // 办公 / 学习
  briefcase: ['M4 8h16v12H4z', 'M9 8V5h6v3', 'M4 13h16'],
  book: ['M12 5c-2-1.5-5-2-8-2v16c3 0 6 .5 8 2 2-1.5 5-2 8-2V3c-3 0-6 .5-8 2z', 'M12 5v16'],
  edit: ['M4 20h16', 'M5 16 15 6l3 3-10 10H5v-3z'],
  // 生活
  home: ['M4 11l8-7 8 7', 'M6 10v10h12V10'],
  cart: ['M4 5h2l2 10h10l2-7H7', 'M10 20h.01', 'M17 20h.01'],
  wallet: ['M4 6h16v13H4z', 'M4 9h16', 'M15 13h.01'],
  chat: ['M4 5h16v11H9l-5 4z'],
  game: ['M3 8h18v9H3z', 'M7 11v3', 'M5.5 12.5h3', 'M16 11.5h.01', 'M18 13.5h.01'],
  // 网络 / 工具
  globe: ['M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z', 'M3 12h18', 'M12 3c2.5 2.5 3.8 5.6 3.8 9s-1.3 6.5-3.8 9c-2.5-2.5-3.8-5.6-3.8-9S9.5 5.5 12 3z'],
  cloud: ['M7 18a4 4 0 0 1 0-8 5.5 5.5 0 0 1 10.7 1.5A3.5 3.5 0 0 1 17 18z'],
  mail: ['M3 6h18v12H3z', 'M3 7l9 6 9-6'],
  rocket: ['M12 2c3.5 2 5 6 5 10l3 4h-6l-2 4-2-4H4l3-4c0-4 1.5-8 5-10z', 'M12 9h.01', 'M9 21h6'],
  // 收藏
  star: ['M12 4l2.3 5.4 5.7.5-4.3 3.8 1.3 5.6L12 16.4 7 19.3l1.3-5.6L4 9.9l5.7-.5L12 4z'],
  heart: ['M12 20l-7-7a4.5 4.5 0 0 1 6.4-6.4L12 7.2l.6-.6a4.5 4.5 0 0 1 6.4 6.4z'],
  bookmark: ['M6 4h12v16l-6-4-6 4z'],
  shield: ['M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6l8-3z', 'M9 12l2 2 4-4'],
  monitor: ['M4 5h16v11H4z', 'M9 20h6', 'M12 16v4'],
}

/** 图标名列表（选择器展示顺序）。 */
export const THEME_ICON_NAMES = Object.keys(THEME_ICONS)

/** 取图标 path 列表；未知名回退 bookmark。 */
export function themeIconPaths(name: string): string[] {
  return THEME_ICONS[name] ?? THEME_ICONS.bookmark
}
