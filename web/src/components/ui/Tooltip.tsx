import type { ReactNode } from "react";
import "./Tooltip.css";

export interface TooltipProps {
  content: ReactNode;
  children: ReactNode;
  visible?: boolean;
}

export function Tooltip({ content, children, visible = true }: TooltipProps) {
  if (!visible || !content) return <>{children}</>;
  return (
    <span className="ui-tooltip-wrap">
      {children}
      <span className="ui-tooltip" role="tooltip">
        {content}
      </span>
    </span>
  );
}
