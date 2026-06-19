import { createPinia, setActivePinia } from 'pinia'

import { useLayoutStore } from '../layout'

describe('useLayoutStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes with default widths and visible sidebars', () => {
    const store = useLayoutStore()

    expect(store.leftSidebarWidth).toBe(280)
    expect(store.rightSidebarWidth).toBe(260)
    expect(store.leftSidebarVisible).toBe(true)
    expect(store.rightSidebarVisible).toBe(true)
  })

  it('toggleLeftSidebar flips left sidebar visibility', () => {
    const store = useLayoutStore()

    store.toggleLeftSidebar()
    expect(store.leftSidebarVisible).toBe(false)

    store.toggleLeftSidebar()
    expect(store.leftSidebarVisible).toBe(true)
  })

  it('toggleRightSidebar flips right sidebar visibility', () => {
    const store = useLayoutStore()

    store.toggleRightSidebar()

    expect(store.rightSidebarVisible).toBe(false)
  })

  it('setLeftSidebarWidth clamps values below the minimum', () => {
    const store = useLayoutStore()

    store.setLeftSidebarWidth(100)

    expect(store.leftSidebarWidth).toBe(180)
  })

  it('setLeftSidebarWidth clamps values above the maximum', () => {
    const store = useLayoutStore()

    store.setLeftSidebarWidth(900)

    expect(store.leftSidebarWidth).toBe(500)
  })

  it('setLeftSidebarWidth keeps values within range unchanged', () => {
    const store = useLayoutStore()

    store.setLeftSidebarWidth(320)

    expect(store.leftSidebarWidth).toBe(320)
  })

  it('setRightSidebarWidth clamps values below the minimum', () => {
    const store = useLayoutStore()

    store.setRightSidebarWidth(100)

    expect(store.rightSidebarWidth).toBe(160)
  })

  it('setRightSidebarWidth clamps values above the maximum', () => {
    const store = useLayoutStore()

    store.setRightSidebarWidth(900)

    expect(store.rightSidebarWidth).toBe(600)
  })
})
