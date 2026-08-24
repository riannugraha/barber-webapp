import "@testing-library/jest-dom";

// Recharts needs ResizeObserver in jsdom — mock minimal untuk <800ms chart tests
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (typeof globalThis.ResizeObserver === "undefined") {
  // @ts-ignore
  globalThis.ResizeObserver = ResizeObserverMock;
}
if (typeof window !== "undefined" && typeof window.ResizeObserver === "undefined") {
  // @ts-ignore
  window.ResizeObserver = ResizeObserverMock;
}
