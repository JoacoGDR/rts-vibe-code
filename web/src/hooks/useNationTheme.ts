import { useEffect } from "react";

/** Applies nation color as --nation-primary on document root while mounted. */
export function useNationTheme(color: string | undefined) {
  useEffect(() => {
    if (!color) return;
    const prev = document.documentElement.style.getPropertyValue("--nation-primary");
    document.documentElement.style.setProperty("--nation-primary", color);
    return () => {
      if (prev) document.documentElement.style.setProperty("--nation-primary", prev);
      else document.documentElement.style.removeProperty("--nation-primary");
    };
  }, [color]);
}
