import "./ResourceChip.css";

export interface ResourceChipProps {
  label: string;
  amount: number;
  rate?: number;
  cap?: number;
  warn?: boolean;
}

export function ResourceChip({ label, amount, rate, cap, warn }: ResourceChipProps) {
  const atCap = cap !== undefined && cap > 0 && amount >= cap * 0.95;
  return (
    <div className={`resource-chip ${warn || atCap ? "resource-chip--warn" : ""}`}>
      <span className="resource-chip__label">{label}</span>
      <span className="resource-chip__value">{Math.floor(amount)}</span>
      {rate !== undefined && (
        <span className={`resource-chip__rate ${rate >= 0 ? "positive" : "negative"}`}>
          {rate >= 0 ? "+" : ""}
          {rate.toFixed(1)}/tick
        </span>
      )}
    </div>
  );
}
