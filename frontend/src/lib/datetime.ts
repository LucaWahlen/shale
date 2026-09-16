export function splitStartsAt(startsAt: string): { date: string; time: string } {
  const [date = "", time = ""] = startsAt.split("T");
  return { date, time };
}

export function todayISO(): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export function intlLocale(language: string): string {
  return language === "de" ? "de-DE" : "en-GB";
}

export function formatDate(dateISO: string, locale: string): string {
  const parsed = new Date(`${dateISO}T12:00`);
  if (Number.isNaN(parsed.getTime())) return dateISO;
  return new Intl.DateTimeFormat(intlLocale(locale), {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(parsed);
}

export function formatDateTime(startsAt: string, locale: string): string {
  const [date = "", time = ""] = startsAt.split("T");
  const parsed = new Date(`${date}T12:00`);
  if (Number.isNaN(parsed.getTime())) return startsAt;
  const day = new Intl.DateTimeFormat(intlLocale(locale), {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(parsed);
  return time ? `${day} · ${time}` : day;
}

export function formatTimeSpan(startsAt: string, endsAt: string, locale: string): string {
  const start = startsAt.slice(11);
  const end = endsAt.slice(11);
  if (endsAt.slice(0, 10) === startsAt.slice(0, 10)) {
    return `${start} – ${end}`;
  }
  const parsed = new Date(`${endsAt.slice(0, 10)}T12:00`);
  if (Number.isNaN(parsed.getTime())) return `${start} – ${end}`;
  const day = new Intl.DateTimeFormat(intlLocale(locale), {
    weekday: "short",
    day: "numeric",
    month: "short",
  }).format(parsed);
  return `${start} – ${day} ${end}`;
}
