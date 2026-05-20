import type { ReactNode } from "react";
import "./Badge.css";

type BadgeVariant = "default" | "active" | "ended" | "waiting";

export interface BadgeProps {
  variant?: BadgeVariant;
  children: ReactNode;
}

export function Badge({ variant = "default", children }: BadgeProps) {
  return <span className={`ui-badge ui-badge--${variant}`}>{children}</span>;
}
