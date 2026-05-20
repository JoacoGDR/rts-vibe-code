import type { ReactNode } from "react";
import "./Panel.css";

export interface PanelProps {
  title?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}

export function Panel({ title, actions, children, className = "" }: PanelProps) {
  return (
    <section className={`ui-panel ${className}`.trim()}>
      {title && (
        <header className="ui-panel__header">
          <h3 className="ui-panel__title">{title}</h3>
          {actions}
        </header>
      )}
      <div className="ui-panel__body">{children}</div>
    </section>
  );
}
