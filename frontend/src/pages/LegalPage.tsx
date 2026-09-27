import { useTranslation } from "react-i18next";

import { AppHeader } from "../components/AppHeader";
import { SiteFooter } from "../components/SiteFooter";
import { useAppSettings } from "../lib/useAppName";

function LegalShell({ title, text }: { title: string; text: string }) {
  const { t } = useTranslation();
  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <AppHeader />
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 py-8 sm:px-6">
        <h1 className="text-2xl font-bold sm:text-3xl">{title}</h1>
        {text.trim() ? (
          <div className="mt-6 whitespace-pre-line text-sm leading-relaxed">{text}</div>
        ) : (
          <p className="mt-6 text-sm text-muted">{t("legal.notConfigured")}</p>
        )}
      </main>
      <SiteFooter />
    </div>
  );
}

export function ImpressumPage() {
  const { t } = useTranslation();
  const { data } = useAppSettings();
  return <LegalShell title={t("common.imprint")} text={data?.imprint_text ?? ""} />;
}

export function PrivacyPage() {
  const { t } = useTranslation();
  const { data } = useAppSettings();
  return <LegalShell title={t("common.privacy")} text={data?.privacy_text ?? ""} />;
}
