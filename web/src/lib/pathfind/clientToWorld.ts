/** Map screen coords → world (viewBox) coords using the transformed layer rect. */
export function clientToWorld(
  clientX: number,
  clientY: number,
  layerRect: DOMRect,
  worldWidth: number,
  worldHeight: number,
): { x: number; y: number } {
  const x = ((clientX - layerRect.left) / layerRect.width) * worldWidth;
  const y = ((clientY - layerRect.top) / layerRect.height) * worldHeight;
  return { x, y };
}
