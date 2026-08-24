import { defineConfig } from "vitest/config";
import path from "node:path";

export default defineConfig({
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
    include: ["**/*.test.tsx", "**/*.test.ts"],
    css: true,
    coverage: {
      provider: "v8",
      reporter: ["text", "lcov", "html"],
      reportsDirectory: "./coverage",
      // PLAN T14: focus on booking+dashboard critical >70%; app pages are E2E covered, not unit
      include: ["components/booking/**/*.{ts,tsx}", "components/dashboard/**/*.{ts,tsx}"],
      exclude: [
        "node_modules/**",
        ".next/**",
        "generated/**",
        "coverage/**",
        "**/*.test.*",
        "**/*.spec.*",
        "e2e/**",
        "vitest.*",
        "playwright.*",
        "next-env.d.ts",
      ],
      thresholds: {
        // Critical: CalendarSlots + BookingForm (booking folder) >70%, dashboard >70%, web >70% for those
        global: {
          lines: 70,
          branches: 55,
          functions: 60,
          statements: 70,
        },
        // Per-folder stricter for booking
        "components/booking/**": {
          lines: 75,
          branches: 55,
          functions: 60,
          statements: 75,
        },
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./"),
    },
  },
});
