import type { ReactNode } from "react";
import { Link } from "react-router";

import { HeaderControls } from "./HeaderControls";
import { useAppName } from "../lib/useAppName";

export function AppHeader({ actions }: { actions?: ReactNode }) {
  const appName = useAppName();
  return (
    <header className="sticky top-0 z-10 transform-gpu border-b border-border/60 bg-background/80 backdrop-blur">
      <div className="mx-auto flex w-full max-w-3xl items-center justify-between gap-3 px-4 py-3 sm:px-6">
        <Link
          to="/"
          className="min-w-0 truncate text-lg font-semibold tracking-tight focus-visible:ring-2 focus-visible:ring-accent"
        >
          {appName}
        </Link>
        <div className="flex shrink-0 items-center gap-1.5">
          <HeaderControls />
          {actions}
        </div>
      </div>
    </header>
  );
}
