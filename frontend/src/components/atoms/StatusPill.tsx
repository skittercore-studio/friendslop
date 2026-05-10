import { JSX } from "preact";

type StatusKind = "done" | "typing" | "idle" | "live" | "waiting";

interface StatusPillProps {
  kind: StatusKind;
  label?: string;
  className?: string;
}

const LABELS: Record<StatusKind, string> = {
  done: "Done",
  typing: "Typing…",
  idle: "Idle",
  live: "Live",
  waiting: "Waiting",
};

/**
 * Compact status chip — used across player rosters to show who's typing,
 * who's submitted, who hasn't started yet, and so on.
 */
export function StatusPill({
  kind,
  label,
  className,
}: StatusPillProps): JSX.Element {
  const text = label ?? LABELS[kind];
  const variant =
    kind === "done"
      ? "fs-chip--on"
      : kind === "live" || kind === "typing"
        ? "fs-chip--live"
        : "";
  const cls = ["fs-chip", variant, className].filter(Boolean).join(" ");
  return <span className={cls}>{text}</span>;
}
