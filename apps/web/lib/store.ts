"use client";

import { create } from "zustand";

type SidebarState = {
  collapsed: boolean;
  toggle: () => void;
  setCollapsed: (v: boolean) => void;
};

export const useSidebarStore = create<SidebarState>((set) => ({
  collapsed: false,
  toggle: () => set((s) => ({ collapsed: !s.collapsed })),
  setCollapsed: (v) => set({ collapsed: v }),
}));

type AppState = {
  orgTimezone: string;
  setOrgTimezone: (tz: string) => void;
};

export const useAppStore = create<AppState>((set) => ({
  orgTimezone: "Asia/Jakarta",
  setOrgTimezone: (tz) => set({ orgTimezone: tz }),
}));
