import { useId } from "react";

const FLAG_CLASS = "h-4 w-auto rounded-[2px] ring-1 ring-border/50";

export function FlagGB() {
  const id = useId();
  const outer = `${id}-outer`;
  const inner = `${id}-inner`;
  return (
    <svg viewBox="0 0 60 30" className={FLAG_CLASS} aria-hidden="true">
      <defs>
        <clipPath id={outer}>
          <path d="M0,0 v30 h60 v-30 z" />
        </clipPath>
        <clipPath id={inner}>
          <path d="M30,15 h30 v15 z v15 h-30 z h-30 v-15 z v-15 h30 z" />
        </clipPath>
      </defs>
      <g clipPath={`url(#${outer})`}>
        <path d="M0,0 v30 h60 v-30 z" fill="#012169" />
        <path d="M0,0 L60,30 M60,0 L0,30" stroke="#fff" strokeWidth="6" />
        <path d="M0,0 L60,30 M60,0 L0,30" clipPath={`url(#${inner})`} stroke="#C8102E" strokeWidth="4" />
        <path d="M30,0 v30 M0,15 h60" stroke="#fff" strokeWidth="10" />
        <path d="M30,0 v30 M0,15 h60" stroke="#C8102E" strokeWidth="6" />
      </g>
    </svg>
  );
}

export function FlagDE() {
  return (
    <svg viewBox="0 0 50 30" className={FLAG_CLASS} aria-hidden="true">
      <rect width="50" height="10" fill="#000" />
      <rect y="10" width="50" height="10" fill="#DD0000" />
      <rect y="20" width="50" height="10" fill="#FFCE00" />
    </svg>
  );
}
