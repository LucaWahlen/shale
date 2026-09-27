import { Link } from "react-router";
import { useTranslation } from "react-i18next";

import { useAppSettings } from "../lib/useAppName";

export function SiteFooter() {
  const { t } = useTranslation();
  const { data } = useAppSettings();
  const appName = data?.app_name ?? "shale";
  const hasImprint = (data?.imprint_text ?? "").trim().length > 0;
  const hasPrivacy = (data?.privacy_text ?? "").trim().length > 0;
  return (
    <footer className="mt-auto border-t border-border/60">
      <div className="mx-auto flex w-full max-w-3xl flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-4 text-sm text-muted sm:px-6">
        <span className="min-w-0 truncate">{appName}</span>
        {hasImprint || hasPrivacy ? (
          <nav className="flex shrink-0 items-center gap-4">
            {hasImprint ? (
              <Link
                to="/impressum"
                className="rounded transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-accent"
              >
                {t("common.imprint")}
              </Link>
            ) : null}
            {hasPrivacy ? (
              <Link
                to="/datenschutz"
                className="rounded transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-accent"
              >
                {t("common.privacy")}
              </Link>
            ) : null}
          </nav>
        ) : null}
      </div>
    </footer>
  );
}
