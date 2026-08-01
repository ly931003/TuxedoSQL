import { defineStore } from 'pinia'

const LEFT_SIDEBAR_MIN = 180
const LEFT_SIDEBAR_MAX = 500
const RIGHT_SIDEBAR_MIN = 160
const RIGHT_SIDEBAR_MAX = 600

const FONT_SCALE_KEY = 'tuxedosql-font-scale'
const FONT_SCALE_MIN = 0.75
const FONT_SCALE_MAX = 1.5
const FONT_SCALE_STEP = 0.05

function loadFontScale(): number {
  try {
    const stored = localStorage.getItem(FONT_SCALE_KEY)
    if (stored) {
      const val = parseFloat(stored)
      if (!Number.isNaN(val)) {
        // 越界值 clamp 并回写，避免每次启动重复解析脏数据
        const clamped = Math.max(FONT_SCALE_MIN, Math.min(FONT_SCALE_MAX, val))
        if (clamped !== val) {
          localStorage.setItem(FONT_SCALE_KEY, String(clamped))
        }
        return clamped
      }
    }
  } catch {
    // localStorage not available (e.g. test environment)
  }
  return 1
}

function applyZoom(scale: number): void {
  try {
    document.documentElement.style.zoom = String(scale)
  } catch {
    // document not available
  }
}

interface LayoutState {
  leftSidebarVisible: boolean
  rightSidebarVisible: boolean
  leftSidebarWidth: number
  rightSidebarWidth: number
  fontScale: number
}

export const useLayoutStore = defineStore('layout', {
  state: (): LayoutState => ({
    leftSidebarVisible: true,
    rightSidebarVisible: true,
    leftSidebarWidth: 280,
    rightSidebarWidth: 260,
    fontScale: loadFontScale(),
  }),

  getters: {
    isMinZoom(state): boolean {
      return state.fontScale <= FONT_SCALE_MIN
    },
    isMaxZoom(state): boolean {
      return state.fontScale >= FONT_SCALE_MAX
    },
  },

  actions: {
    toggleLeftSidebar(): void {
      this.leftSidebarVisible = !this.leftSidebarVisible
    },
    toggleRightSidebar(): void {
      this.rightSidebarVisible = !this.rightSidebarVisible
    },
    setLeftSidebarWidth(width: number): void {
      this.leftSidebarWidth = Math.max(LEFT_SIDEBAR_MIN, Math.min(LEFT_SIDEBAR_MAX, width))
    },
    setRightSidebarWidth(width: number): void {
      this.rightSidebarWidth = Math.max(RIGHT_SIDEBAR_MIN, Math.min(RIGHT_SIDEBAR_MAX, width))
    },
    setFontScale(scale: number): void {
      this.fontScale = Math.max(FONT_SCALE_MIN, Math.min(FONT_SCALE_MAX, scale))
      try {
        applyZoom(this.fontScale)
        localStorage.setItem(FONT_SCALE_KEY, String(this.fontScale))
      } catch {
        // localStorage not available (e.g. test environment)
      }
    },
    zoomIn(): void {
      this.setFontScale(this.fontScale + FONT_SCALE_STEP)
    },
    zoomOut(): void {
      this.setFontScale(this.fontScale - FONT_SCALE_STEP)
    },
    resetZoom(): void {
      this.setFontScale(1)
    },
  },
})
