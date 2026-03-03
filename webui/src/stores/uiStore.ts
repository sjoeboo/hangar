import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark' | 'system'

interface UIState {
  selectedSessionId: string | null
  sidebarOpen: boolean
  sidebarWidth: number
  theme: Theme
  setSelectedSession: (id: string | null) => void
  setSidebarOpen: (open: boolean) => void
  setSidebarWidth: (w: number) => void
  setTheme: (theme: Theme) => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      selectedSessionId: null,
      sidebarOpen: true,
      sidebarWidth: 240,
      theme: 'dark',
      setSelectedSession: (id) => set({ selectedSessionId: id }),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setSidebarWidth: (w) => set({ sidebarWidth: w }),
      setTheme: (theme) => set({ theme }),
    }),
    { name: 'hangar-ui' }
  )
)
