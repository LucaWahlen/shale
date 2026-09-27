import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Button, Card, Dropdown } from "@heroui/react";

import { api } from "../api/client";
import type { AuditEntry, AuditListParams } from "../api/client";
import { intlLocale } from "../lib/datetime";

const PAGE_SIZE = 25;

const ACTION_OPTIONS: { value: string; labelKey: string }[] = [
  { value: "", labelKey: "admin.audit.actionAll" },
  { value: "attendee.added", labelKey: "admin.audit.actions.attendeeAdded" },
  { value: "attendee.removed", labelKey: "admin.audit.actions.attendeeRemoved" },
  { value: "schedule.created", labelKey: "admin.audit.actions.scheduleCreated" },
  { value: "schedule.updated", labelKey: "admin.audit.actions.scheduleUpdated" },
  { value: "schedule.deleted", labelKey: "admin.audit.actions.scheduleDeleted" },
  { value: "schedule.duplicated", labelKey: "admin.audit.actions.scheduleDuplicated" },
  { value: "event.created", labelKey: "admin.audit.actions.eventCreated" },
  { value: "event.updated", labelKey: "admin.audit.actions.eventUpdated" },
  { value: "event.deleted", labelKey: "admin.audit.actions.eventDeleted" },
];

function ChevronDownIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="size-4" aria-hidden="true">
      <path d="m6 9 6 6 6-6" />
    </svg>
  );
}

function formatTimestamp(iso: string, locale: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return new Intl.DateTimeFormat(intlLocale(locale), {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(d);
}

function actionLabelKey(action: string): string {
  return ACTION_OPTIONS.find((o) => o.value === action)?.labelKey ?? action;
}

function EntryRow({ entry }: { entry: AuditEntry }) {
  const { t, i18n } = useTranslation();
  return (
    <li className="flex flex-col gap-0.5 py-2.5">
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5">
        <span className="text-sm font-medium">
          {t(actionLabelKey(entry.action))}
          {entry.actor_name ? ` · ${entry.actor_name}` : ""}
        </span>
        <span className="text-xs tabular-nums text-muted">
          {formatTimestamp(entry.created_at, i18n.language)}
        </span>
      </div>
      <p className="text-xs text-muted">
        <span>{entry.actor === "admin" ? t("admin.audit.actor.admin") : t("admin.audit.actor.public")}</span>
        {entry.event_name ? (
          <>
            <span aria-hidden="true"> · </span>
            <span>{entry.event_name}</span>
          </>
        ) : null}
        {entry.schedule_title ? (
          <>
            <span aria-hidden="true"> · </span>
            <span>{entry.schedule_title}</span>
          </>
        ) : null}
        {entry.detail ? (
          <>
            <span aria-hidden="true"> · </span>
            <span>{entry.detail}</span>
          </>
        ) : null}
      </p>
    </li>
  );
}

export function AdminAudit() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const action = searchParams.get("action") ?? "";
  const scheduleId = searchParams.get("schedule") ?? "";
  const page = Math.max(1, Number.parseInt(searchParams.get("page") ?? "1", 10) || 1);

  const [searchInput, setSearchInput] = useState(q);
  useEffect(() => {
    setSearchInput(q);
  }, [q]);

  const updateParams = (next: Record<string, string | undefined>) => {
    setSearchParams(
      (prev) => {
        const params = new URLSearchParams(prev);
        for (const [key, value] of Object.entries(next)) {
          if (value === undefined || value === "") params.delete(key);
          else params.set(key, value);
        }
        return params;
      },
      { replace: true },
    );
  };

  useEffect(() => {
    if (searchInput === q) return;
    const id = window.setTimeout(() => {
      updateParams({ q: searchInput || undefined, page: undefined });
    }, 300);
    return () => window.clearTimeout(id);
  }, [searchInput, q]);

  const { data: scheduleOptions } = useQuery({
    queryKey: ["admin-schedules", "audit-filter"],
    queryFn: () => api.listSchedules({ pageSize: 100, includePast: true }),
  });

  const params = useMemo<AuditListParams>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      action: action || undefined,
      scheduleId: scheduleId || undefined,
      q: q.trim() || undefined,
    }),
    [page, action, scheduleId, q],
  );

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ["admin-audit", params],
    queryFn: () => api.listAudit(params),
    placeholderData: keepPreviousData,
  });

  const total = data?.total ?? 0;
  const pageSize = data?.page_size ?? PAGE_SIZE;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const items = data?.items ?? [];
  const currentAction = ACTION_OPTIONS.find((o) => o.value === action) ?? ACTION_OPTIONS[0];
  const currentSchedule =
    scheduleOptions?.items.find((s) => s.id === scheduleId)?.title ?? "";

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-xl font-semibold">{t("admin.audit.title")}</h1>
          <p className="text-sm text-muted">{t("admin.audit.subtitle")}</p>
        </div>
        {scheduleId ? (
          <Button variant="secondary" size="sm" onPress={() => updateParams({ schedule: undefined, page: undefined })}>
            {t("admin.audit.clearSchedule")}
          </Button>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <input
          type="search"
          name="audit-search"
          autoComplete="off"
          className="min-w-40 flex-1 rounded-lg border border-border bg-surface px-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-accent"
          placeholder={t("admin.audit.searchPlaceholder")}
          aria-label={t("admin.audit.searchPlaceholder")}
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
        />
        <Dropdown>
          <Dropdown.Trigger
            aria-label={t("admin.audit.actionLabel")}
            style={{ display: "inline-flex", transform: "none" }}
            className="items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-foreground transition-colors hover:bg-surface-secondary"
          >
            <span className="font-medium">{t(currentAction.labelKey)}</span>
            <ChevronDownIcon />
          </Dropdown.Trigger>
          <Dropdown.Popover placement="bottom end">
            <Dropdown.Menu
              onAction={(key) =>
                updateParams({ action: key === "all" ? undefined : String(key), page: undefined })
              }
            >
              {ACTION_OPTIONS.map((o) => (
                <Dropdown.Item id={o.value || "all"} key={o.value || "all"}>
                  {t(o.labelKey)}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          </Dropdown.Popover>
        </Dropdown>
        <Dropdown>
          <Dropdown.Trigger
            aria-label={t("admin.audit.scheduleLabel")}
            style={{ display: "inline-flex", transform: "none" }}
            className="max-w-52 items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-foreground transition-colors hover:bg-surface-secondary"
          >
            <span className="min-w-0 truncate font-medium">
              {currentSchedule || t("admin.audit.scheduleAll")}
            </span>
            <ChevronDownIcon />
          </Dropdown.Trigger>
          <Dropdown.Popover placement="bottom end">
            <Dropdown.Menu
              onAction={(key) =>
                updateParams({ schedule: key === "all" ? undefined : String(key), page: undefined })
              }
            >
              <Dropdown.Item id="all" key="all">
                {t("admin.audit.scheduleAll")}
              </Dropdown.Item>
              {(scheduleOptions?.items ?? []).map((s) => (
                <Dropdown.Item id={s.id} key={s.id}>
                  {s.title}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          </Dropdown.Popover>
        </Dropdown>
      </div>

      {isLoading ? <p className="text-muted">{t("common.loading")}</p> : null}

      {data && total === 0 ? (
        <Card>
          <Card.Content className="py-10 text-center text-sm text-muted">
            {t("admin.audit.empty")}
          </Card.Content>
        </Card>
      ) : null}

      {items.length > 0 ? (
        <Card>
          <Card.Content>
            <ul className="flex flex-col divide-y divide-border/60">
              {items.map((entry) => (
                <EntryRow key={entry.id} entry={entry} />
              ))}
            </ul>
          </Card.Content>
        </Card>
      ) : null}

      {data && totalPages > 1 ? (
        <nav className="flex items-center justify-between gap-2" aria-label={t("admin.audit.pagination")}>
          <Button
            size="sm"
            variant="secondary"
            isDisabled={page <= 1 || isFetching}
            onPress={() => updateParams({ page: String(page - 1) })}
          >
            {t("admin.schedules.prevPage")}
          </Button>
          <span className="text-sm text-muted" aria-live="polite">
            {t("admin.schedules.pageOf", { page, pages: totalPages })}
          </span>
          <Button
            size="sm"
            variant="secondary"
            isDisabled={page >= totalPages || isFetching}
            onPress={() => updateParams({ page: String(page + 1) })}
          >
            {t("admin.schedules.nextPage")}
          </Button>
        </nav>
      ) : null}
    </div>
  );
}
