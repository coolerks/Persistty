export function isVerticalWheel(deltaX: number, deltaY: number): boolean {
  return Number.isFinite(deltaX) && Number.isFinite(deltaY) && deltaY !== 0 && Math.abs(deltaY) >= Math.abs(deltaX);
}

// Preserve sub-pixel trackpad deltas; only normalize units while a snapshot loads.
export function wheelPixels(deltaY: number, deltaMode: number, lineHeight: number, height: number): number {
  if (!Number.isFinite(deltaY) || deltaY === 0) return 0;
  return deltaY * (deltaMode === 1 ? lineHeight : deltaMode === 2 ? height : 1);
}
