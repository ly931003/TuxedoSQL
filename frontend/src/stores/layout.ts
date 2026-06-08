import { defineStore } from 'pinia'

const LEFT_SIDEBAR_MIN = 180
const LEFT_SIDEBAR_MAX = 500
const RIGHT_SIDEBAR_MIN = 160
const RIGHT_SIDEBAR_MAX = 600

interface LayoutState {
  leftSidebarVisible: boolean
  rightSidebarVisible: boolean
  leftSidebarWidth: number
  rightSidebarWidth: number
}

export const useLayoutStore = defineStore('layout', {
  state: (): LayoutState => ({
    leftSidebarVisible: true,
    rightSidebarVisible: true,
    leftSidebarWidth: 280,
    rightSidebarWidth: 260,
  }),

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
  },
})