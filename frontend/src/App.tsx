import { useEffect } from "react";
import { Navigate, Route, Routes } from "react-router";
import { useTranslation } from "react-i18next";
import { I18nProvider, ToastProvider } from "@heroui/react";

import { changeLanguage, storedLocale } from "./i18n";
import { useAppSettings } from "./lib/useAppName";
import { AdminLayout } from "./pages/AdminLayout";
import { AdminSchedules } from "./pages/AdminSchedules";
import { AdminScheduleEditor } from "./pages/AdminScheduleEditor";
import { AdminSettings } from "./pages/AdminSettings";
import { AdminAudit } from "./pages/AdminAudit";
import { SchedulePage } from "./pages/SchedulePage";
import { AdminLogin } from "./pages/AdminLogin";
import { ImpressumPage, PrivacyPage } from "./pages/LegalPage";

export default function App() {
  const { i18n } = useTranslation();
  const { data: settings } = useAppSettings();

  useEffect(() => {
    if (settings && !storedLocale()) {
      changeLanguage(settings.default_language);
    }
  }, [settings]);

  useEffect(() => {
    if (settings) {
      document.title = settings.app_name;
    }
  }, [settings]);

  if (!settings) {
    return null;
  }

  return (
    <>
      { }
      <I18nProvider locale={i18n.language === "de" ? "de-DE" : "en-GB"}>
        <Routes>
          <Route path="/" element={<AdminLayout />}>
            <Route index element={<AdminSchedules />} />
            <Route path="schedules/:id" element={<AdminScheduleEditor />} />
            <Route path="settings" element={<AdminSettings />} />
            <Route path="audit" element={<AdminAudit />} />
          </Route>
        <Route path="/s/:id" element={<SchedulePage />} />
        <Route path="/impressum" element={<ImpressumPage />} />
        <Route path="/datenschutz" element={<PrivacyPage />} />
        <Route path="/login" element={<AdminLogin />} />
        <Route path="/admin/login" element={<Navigate to="/login" replace />} />
        <Route path="*" element={<AdminLayout />}>
          <Route path="*" element={<AdminSchedules />} />
        </Route>
        </Routes>
      </I18nProvider>
      <ToastProvider />
    </>
  );
}
