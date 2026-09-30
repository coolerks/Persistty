import { useEffect, useLayoutEffect, useRef } from "react";

// Portals bubble React events through their owner, not their current DOM host.
export function useDropTarget(types: string[], onDrop: (event: DragEvent) => void) {
  const ref = useRef<HTMLElement>(null);
  const handler = useRef({ types, onDrop });
  useLayoutEffect(() => { handler.current = { types, onDrop }; }, [types, onDrop]);
  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    const capture = (event: DragEvent) => {
      if (!event.dataTransfer || !handler.current.types.some(type => event.dataTransfer?.types.includes(type))) return;
      event.preventDefault(); event.stopPropagation();
      if (event.type === "drop") handler.current.onDrop(event);
    };
    element.addEventListener("dragover", capture, true);
    element.addEventListener("drop", capture, true);
    return () => { element.removeEventListener("dragover", capture, true); element.removeEventListener("drop", capture, true); };
  }, []);
  return ref;
}
