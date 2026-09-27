import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Card, Modal, Radio, RadioGroup, toast, useOverlayState } from "@heroui/react";

import { api } from "../api/client";
import type { AppSettings } from "../api/client";
import { apiErrorMessage } from "../lib/errors";
import { FlagDE, FlagGB } from "../components/flags";
import { LOCALE_NAMES, locales, type Locale } from "../i18n";

export function AdminSettings() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const confirmModal = useOverlayState();
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [appName, setAppName] = useState("");
  const [language, setLanguage] = useState<AppSettings["default_language"]>("en");
  const [imprintText, setImprintText] = useState("");
  const [privacyText, setPrivacyText] = useState("");

  const { data: settings } = useQuery({
    queryKey: ["admin-settings"],
    queryFn: api.getAdminSettings,
  });

  useEffect(() => {
    if (settings) {
      setAppName(settings.app_name);
      setLanguage(settings.default_language);
      setImprintText(settings.imprint_text);
      setPrivacyText(settings.privacy_text);
    }
  }, [settings]);

  const settingsDirty =
    settings !== undefined &&
    (appName.trim() !== settings.app_name ||
      language !== settings.default_language ||
      imprintText.trim() !== settings.imprint_text ||
      privacyText.trim() !== settings.privacy_text);

  const saveSettings = useMutation({
    mutationFn: () =>
      api.putAdminSettings({
        app_name: appName.trim(),
        default_language: language,
        imprint_text: imprintText.trim(),
        privacy_text: privacyText.trim(),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin-settings"] });
      void queryClient.invalidateQueries({ queryKey: ["public-settings"] });
      toast.success(t("admin.editor.saved"));
    },
    onError: (err) => toast.danger(apiErrorMessage(err, t)),
  });

  const importMutation = useMutation({
    mutationFn: async (file: File) => api.import(await file.text()),
    onSuccess: (res) => {
      void queryClient.invalidateQueries();
      toast.success(
        t("admin.settings.importSuccess", {
          schedules: res.imported.schedules,
          events: res.imported.events,
          attendees: res.imported.attendees,
        }),
      );
    },
    onError: (err) => toast.danger(apiErrorMessage(err, t)),
    onSettled: () => {
      setPendingFile(null);
      confirmModal.close();
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-xl font-semibold">{t("admin.settings.title")}</h1>

      <Card>
        <Card.Content className="gap-4">
          <div>
            <h2 className="text-base font-semibold">{t("admin.settings.generalTitle")}</h2>
          </div>
          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">{t("admin.settings.appNameLabel")}</span>
            <input
              name="app-name"
              autoComplete="off"
              className="w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
              value={appName}
              placeholder={t("admin.settings.appNamePlaceholder")}
              onChange={(e) => setAppName(e.target.value)}
            />
            <span className="text-xs text-muted">{t("admin.settings.appNameHint")}</span>
          </label>
          <div>
            <h2 className="text-base font-semibold">{t("admin.settings.languageTitle")}</h2>
            <p className="text-sm text-muted">{t("admin.settings.languageHint")}</p>
          </div>
          <RadioGroup
            name="default-language"
            value={language}
            onChange={(v) => setLanguage(v as AppSettings["default_language"])}
          >
            {locales.map((lang) => (
              <Radio key={lang} value={lang}>
                <Radio.Content>
                  <Radio.Control>
                    <Radio.Indicator />
                  </Radio.Control>
                  <span className="inline-flex items-center gap-2 text-sm font-medium">
                    {lang === "en" ? <FlagGB /> : <FlagDE />}
                    {LOCALE_NAMES[lang as Locale]}
                  </span>
                </Radio.Content>
              </Radio>
            ))}
          </RadioGroup>
        </Card.Content>
      </Card>

      <Card>
        <Card.Content className="gap-4">
          <div>
            <h2 className="text-base font-semibold">{t("admin.settings.legalTitle")}</h2>
            <p className="text-sm text-muted">{t("admin.settings.legalHint")}</p>
          </div>
          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">{t("admin.settings.imprintLabel")}</span>
            <textarea
              name="imprint-text"
              className="min-h-32 w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
              value={imprintText}
              onChange={(e) => setImprintText(e.target.value)}
            />
            <span className="text-xs text-muted">{t("admin.settings.imprintHint")}</span>
          </label>
          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">{t("admin.settings.privacyLabel")}</span>
            <textarea
              name="privacy-text"
              className="min-h-32 w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
              value={privacyText}
              onChange={(e) => setPrivacyText(e.target.value)}
            />
            <span className="text-xs text-muted">{t("admin.settings.privacyHint")}</span>
          </label>
          <div>
            <Button
              isDisabled={saveSettings.isPending || !settings || !settingsDirty || appName.trim().length === 0}
              onPress={() => saveSettings.mutate()}
            >
              {t("common.save")}
            </Button>
          </div>
        </Card.Content>
      </Card>

      <Card>
        <Card.Content className="gap-4">
          <h2 className="text-base font-semibold">{t("admin.settings.dataTitle")}</h2>

          <div>
            <h3 className="text-sm font-semibold">{t("admin.settings.exportTitle")}</h3>
            <p className="text-sm text-muted">{t("admin.settings.exportHint")}</p>
            <div className="mt-2">
              <a href={api.exportUrl} download>
                <Button variant="secondary">{t("admin.settings.exportButton")}</Button>
              </a>
            </div>
          </div>

          <div>
            <h3 className="text-sm font-semibold">{t("admin.settings.importTitle")}</h3>
            <p className="text-sm text-muted">{t("admin.settings.importHint")}</p>
            <div className="mt-2">
              <input
                ref={fileRef}
                type="file"
                name="import-file"
                accept="application/json,.json"
                className="sr-only"
                tabIndex={-1}
                aria-hidden="true"
                onChange={(e) => {
                  const file = e.target.files?.[0] ?? null;
                  e.target.value = "";
                  if (!file) return;
                  setPendingFile(file);
                  confirmModal.open();
                }}
              />
              <Button variant="danger" onPress={() => fileRef.current?.click()}>
                {t("admin.settings.importButton")}
              </Button>
            </div>
          </div>
        </Card.Content>
      </Card>

      <Modal.Backdrop isOpen={confirmModal.isOpen} onOpenChange={confirmModal.setOpen}>
        <Modal.Container size="sm">
          <Modal.Dialog>
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{t("admin.settings.importConfirmTitle")}</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <p className="text-sm text-muted">
                {t("admin.settings.importConfirmBody", { file: pendingFile?.name ?? "" })}
              </p>
            </Modal.Body>
            <Modal.Footer>
              <Button slot="close" variant="secondary">{t("common.cancel")}</Button>
              <Button
                variant="danger"
                isDisabled={importMutation.isPending || !pendingFile}
                onPress={() => pendingFile && importMutation.mutate(pendingFile)}
              >
                {t("admin.settings.importButton")}
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </div>
  );
}
