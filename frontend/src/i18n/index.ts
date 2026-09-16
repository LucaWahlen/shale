import i18next from "i18next";
import { initReactI18next } from "react-i18next";

import en from "./en.json";
import de from "./de.json";

export const locales = ["en", "de"] as const;
export type Locale = (typeof locales)[number];

export const LOCALE_NAMES: Record<Locale, string> = { en: "English", de: "Deutsch" };

const LANGUAGE_KEY = "shale.language";

export function storedLocale(): Locale | null {
  try {
    const v = localStorage.getItem(LANGUAGE_KEY);
    return v === "en" || v === "de" ? v : null;
  } catch {
    return null;
  }
}

function applyDocumentLanguage(locale: Locale): void {
  document.documentElement.lang = locale;
}

export function initI18n(defaultLanguage: string): typeof i18next {
  const initial: Locale = storedLocale() ?? (defaultLanguage === "de" ? "de" : "en");
  if (!i18next.isInitialized) {
    void i18next.use(initReactI18next).init({
      resources: {
        en: { translation: en },
        de: { translation: de },
      },
      lng: initial,
      fallbackLng: "en",
      interpolation: { escapeValue: false },
      returnNull: false,
      react: { useSuspense: false },
    });
  }
  applyDocumentLanguage(initial);
  return i18next;
}

export function changeLanguage(locale: Locale): typeof i18next {
  if (!i18next.isInitialized) {
    return initI18n(locale);
  }
  if (i18next.language !== locale) {
    void i18next.changeLanguage(locale);
  }
  applyDocumentLanguage(locale);
  return i18next;
}

export function currentLocale(): Locale {
  return i18next.language === "de" ? "de" : "en";
}
