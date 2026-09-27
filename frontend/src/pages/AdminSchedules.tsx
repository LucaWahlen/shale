import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Card, Chip, Dropdown, Modal, toast, useOverlayState } from "@heroui/react";

import { api } from "../api/client";
import type { AdminSchedule, ScheduleListParams, ScheduleSort } from "../api/client";
import { apiErrorMessage } from "../lib/errors";
import { formatDateTime, formatTimeSpan, intlLocale } from "../lib/datetime";

const PAGE_SIZE = 10;

const SORT_OPTIONS: { value: ScheduleSort; labelKey: string }[] = [
  { value: "newest", labelKey: "admin.schedules.sortNewest" },
  { value: "oldest", labelKey: "admin.schedules.sortOldest" },
  { value: "updated", labelKey: "admin.schedules.sortUpdated" },
  { value: "soonest", labelKey: "admin.schedules.sortSoonest" },
  { value: "events", labelKey: "admin.schedules.sortEvents" },
  { value: "title", labelKey: "admin.schedules.sortTitleAsc" },
  { value: "title_desc", labelKey: "admin.schedules.sortTitleDesc" },
];

function DotsIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className="size-5" aria-hidden="true">
      <circle cx="5" cy="12" r="1.8" />
      <circle cx="12" cy="12" r="1.8" />
      <circle cx="19" cy="12" r="1.8" />
    </svg>
  );
}

function ChevronDownIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="size-4" aria-hidden="true">
      <path d="m6 9 6 6 6-6" />
    </svg>
  );
}

function HistoryIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="size-4" aria-hidden="true">
      <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
      <path d="M3 3v5h5" />
      <path d="M12 7v5l4 2" />
    </svg>
  );
}

function whenRange(schedule: AdminSchedule, locale: string): string | null {
  if (!schedule.first_starts_at) return null;
  const fmt = (startsAt: string) => {
    const parsed = new Date(`${startsAt.slice(0, 10)}T12:00`);
    if (Number.isNaN(parsed.getTime())) return null;
    const day = new Intl.DateTimeFormat(intlLocale(locale), {
      day: "numeric",
      month: "short",
      year: "numeric",
    }).format(parsed);
    return `${day}, ${startsAt.slice(11)}`;
  };
  const first = fmt(schedule.first_starts_at);
  if (!first) return null;
  const last =
    schedule.last_starts_at && schedule.last_starts_at !== schedule.first_starts_at
      ? fmt(schedule.last_starts_at)
      : null;
  return last ? `${first} – ${last}` : first;
}

export function AdminSchedules() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createModal = useOverlayState();
  const deleteModal = useOverlayState();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [target, setTarget] = useState<AdminSchedule | null>(null);

  const [searchParams, setSearchParams] = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const sortParam = searchParams.get("sort");
  const sort: ScheduleSort = SORT_OPTIONS.some((o) => o.value === sortParam)
    ? (sortParam as ScheduleSort)
    : "newest";
  const page = Math.max(1, Number.parseInt(searchParams.get("page") ?? "1", 10) || 1);
  const includePast = searchParams.get("history") === "1";

  const [searchInput, setSearchInput] = useState(q);

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
    setSearchInput(q);
  }, [q]);

  useEffect(() => {
    if (searchInput === q) return;
    const id = window.setTimeout(() => {
      updateParams({ q: searchInput || undefined, page: undefined });
    }, 300);
    return () => window.clearTimeout(id);
  }, [searchInput, q]);

  const params = useMemo<ScheduleListParams>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      sort,
      q: q.trim() || undefined,
      includePast,
    }),
    [page, sort, q, includePast],
  );

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ["admin-schedules", params],
    queryFn: () => api.listSchedules(params),
    placeholderData: keepPreviousData,
  });

  const resetForm = () => {
    setTitle("");
    setDescription("");
  };

  const create = useMutation({
    mutationFn: () => api.createSchedule({ title, description }),
    onSuccess: (created) => {
      void queryClient.invalidateQueries({ queryKey: ["admin-schedules"] });
      createModal.close();
      resetForm();
      navigate(`/schedules/${created.id}`);
    },
    onError: (err) => {
      toast.danger(apiErrorMessage(err, t));
    },
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.deleteSchedule(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin-schedules"] });
      deleteModal.close();
      toast.success(t("admin.schedules.deleted"));
    },
    onError: () => toast.danger(t("errors.unexpected")),
  });

  const total = data?.total ?? 0;
  const pageSize = data?.page_size ?? PAGE_SIZE;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const hasQuery = q.trim().length > 0;
  const items = data?.items ?? [];
  const currentSort = SORT_OPTIONS.find((o) => o.value === sort) ?? SORT_OPTIONS[0];

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-xl font-semibold">{t("admin.schedules.title")}</h1>
          {data && total > 0 ? (
            <p className="text-sm text-muted" aria-live="polite">
              {total <= pageSize && page === 1
                ? t("admin.schedules.count", { count: total })
                : t("admin.schedules.showing", {
                    from: (page - 1) * pageSize + 1,
                    to: (page - 1) * pageSize + items.length,
                    total,
                  })}
            </p>
          ) : null}
        </div>
        <Button onPress={() => { resetForm(); createModal.open(); }}>{t("admin.schedules.create")}</Button>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <input
          type="search"
          name="schedule-search"
          autoComplete="off"
          className="min-w-40 flex-1 rounded-lg border border-border bg-surface px-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-accent"
          placeholder={t("admin.schedules.searchPlaceholder")}
          aria-label={t("admin.schedules.searchPlaceholder")}
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
        />
        <Dropdown>
          <Dropdown.Trigger
            aria-label={t("admin.schedules.sortLabel")}
            style={{ display: "inline-flex", transform: "none" }}
            className="items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-foreground transition-colors hover:bg-surface-secondary"
          >
            <span className="text-muted">{t("admin.schedules.sortLabel")}</span>
            <span className="font-medium">{t(currentSort.labelKey)}</span>
            <ChevronDownIcon />
          </Dropdown.Trigger>
          <Dropdown.Popover placement="bottom end">
            <Dropdown.Menu
              onAction={(key) =>
                updateParams({
                  sort: String(key) === "newest" ? undefined : String(key),
                  page: undefined,
                })
              }
            >
              {SORT_OPTIONS.map((o) => (
                <Dropdown.Item id={o.value} key={o.value}>
                  {t(o.labelKey)}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          </Dropdown.Popover>
        </Dropdown>
        <Button
          size="sm"
          isIconOnly
          variant={includePast ? "primary" : "secondary"}
          aria-label={t("admin.schedules.history")}
          aria-pressed={includePast}
          onPress={() =>
            updateParams({ history: includePast ? undefined : "1", page: undefined })
          }
        >
          <HistoryIcon />
        </Button>
      </div>

      {isLoading ? <p className="text-muted">{t("common.loading")}</p> : null}

      {data && total === 0 ? (
        <Card>
          <Card.Content className="flex flex-col items-center gap-3 py-12 text-center">
            {hasQuery ? (
              <>
                <p className="font-medium">{t("admin.schedules.noResultsTitle")}</p>
                <p className="max-w-sm text-sm text-muted">
                  {t("admin.schedules.noResults", { query: q })}
                </p>
                <Button
                  variant="secondary"
                  onPress={() => {
                    setSearchInput("");
                    updateParams({ q: undefined, page: undefined });
                  }}
                >
                  {t("admin.schedules.clearSearch")}
                </Button>
              </>
            ) : !includePast ? (
              <>
                <p className="font-medium">{t("admin.schedules.emptyUpcomingTitle")}</p>
                <p className="max-w-sm text-sm text-muted">{t("admin.schedules.emptyUpcoming")}</p>
                <div className="flex flex-wrap justify-center gap-2">
                  <Button
                    variant="secondary"
                    onPress={() => updateParams({ history: "1", page: undefined })}
                  >
                    {t("admin.schedules.showPast")}
                  </Button>
                  <Button onPress={() => { resetForm(); createModal.open(); }}>
                    {t("admin.schedules.create")}
                  </Button>
                </div>
              </>
            ) : (
              <>
                <p className="font-medium">{t("admin.schedules.emptyTitle")}</p>
                <p className="max-w-sm text-sm text-muted">{t("admin.schedules.empty")}</p>
                <Button variant="secondary" onPress={() => { resetForm(); createModal.open(); }}>
                  {t("admin.schedules.create")}
                </Button>
              </>
            )}
          </Card.Content>
        </Card>
      ) : null}

      <div className="flex flex-col gap-3">
        {items.map((s) => {
          const events = s.events ?? [];
          const attendeeTotal = events.reduce((n, ev) => n + ev.attendees.length, 0);
          const range = whenRange(s, i18n.language);
          const shareURL = `${window.location.origin}/s/${s.id}`;
          const copyLink = async () => {
            try {
              await navigator.clipboard.writeText(shareURL);
              toast.success(t("admin.editor.linkCopied"));
            } catch {
              window.prompt(t("admin.editor.shareUrlLabel"), shareURL);
            }
          };
          return (
            <Card key={s.id} className={s.is_past ? "opacity-75" : undefined}>
              <Card.Content className="gap-2.5">
                <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
                  <div className="min-w-0 flex-1">
                    <div className="flex min-w-0 items-center gap-2">
                      <Link
                        to={`/s/${s.id}`}
                        className="min-w-0 truncate text-base font-semibold focus-visible:ring-2 focus-visible:ring-accent"
                      >
                        {s.title}
                      </Link>
                      {s.is_past ? (
                        <Chip variant="soft" size="sm">
                          {t("admin.schedules.pastBadge")}
                        </Chip>
                      ) : null}
                    </div>
                    {s.description ? (
                      <p className="mt-0.5 whitespace-pre-line text-sm text-muted">{s.description}</p>
                    ) : null}
                  </div>
                  <div className="flex shrink-0 items-center gap-1.5">
                    <Button size="sm" variant="secondary" onPress={() => navigate(`/s/${s.id}`)}>
                      {t("admin.schedules.open")}
                    </Button>
                    <Button size="sm" variant="secondary" onPress={() => navigate(`/schedules/${s.id}`)}>
                      {t("common.edit")}
                    </Button>
                    <Dropdown>
                      <Dropdown.Trigger
                        aria-label={t("common.actions")}
                        className="inline-grid size-8 place-items-center rounded-full text-muted transition-colors hover:bg-surface-secondary hover:text-foreground"
                      >
                        <DotsIcon />
                      </Dropdown.Trigger>
                      <Dropdown.Popover placement="bottom end">
                        <Dropdown.Menu
                          onAction={(key) => {
                            if (key === "copy-link") void copyLink();
                            if (key === "delete") {
                              setTarget(s);
                              deleteModal.open();
                            }
                          }}
                        >
                          <Dropdown.Item id="copy-link">{t("admin.editor.copyLink")}</Dropdown.Item>
                          <Dropdown.Item id="delete" variant="danger">
                            {t("common.delete")}
                          </Dropdown.Item>
                        </Dropdown.Menu>
                      </Dropdown.Popover>
                    </Dropdown>
                  </div>
                </div>

                <p className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-sm text-muted">
                  <span>{t("admin.schedules.events", { count: events.length })}</span>
                  <span aria-hidden="true">·</span>
                  <span>{t("admin.schedules.attendees", { count: attendeeTotal })}</span>
                  {range ? (
                    <>
                      <span aria-hidden="true">·</span>
                      <span className="tabular-nums">{range}</span>
                    </>
                  ) : null}
                </p>

                {events.length > 0 ? (
                  <ul className="flex flex-col divide-y divide-border/60 border-t border-border/60">
                    {events.map((ev) => {
                      const timeLabel = ev.all_day
                        ? t("schedule.allDay")
                        : ev.ends_at
                          ? formatTimeSpan(ev.starts_at, ev.ends_at, i18n.language)
                          : ev.starts_at.slice(11);
                      return (
                        <li key={ev.id} className="flex flex-col gap-0.5 py-2">
                          <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5">
                            <span className="min-w-0 truncate text-sm font-medium">{ev.name}</span>
                            <span className="text-xs tabular-nums text-muted">
                              {formatDateTime(ev.starts_at, i18n.language)}
                              {ev.all_day || timeLabel !== "" ? ` · ${timeLabel}` : ""}
                              {ev.location ? ` · ${ev.location}` : ""}
                            </span>
                          </div>
                          <p className="truncate text-xs text-muted">
                            {ev.attendees.length > 0
                              ? `${ev.attendees.map((a) => a.name).join(", ")} · ${t("schedule.attendees", { count: ev.attendees.length })}`
                              : t("admin.editor.attendeesEmpty")}
                          </p>
                        </li>
                      );
                    })}
                  </ul>
                ) : (
                  <p className="text-sm text-muted">{t("admin.schedules.noDates")}</p>
                )}
              </Card.Content>
            </Card>
          );
        })}
      </div>

      {data && totalPages > 1 ? (
        <nav className="flex items-center justify-between gap-2" aria-label={t("admin.schedules.pagination")}>
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

      <Modal.Backdrop isOpen={createModal.isOpen} onOpenChange={createModal.setOpen}>
        <Modal.Container size="sm">
          <Modal.Dialog>
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{t("admin.schedules.createTitle")}</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <form id="create-schedule" className="flex flex-col gap-4" onSubmit={(e) => { e.preventDefault(); create.mutate(); }}>
                <label className="flex flex-col gap-1.5">
                  <span className="text-sm font-medium">{t("admin.schedules.titleLabel")}</span>
                  <input
                    name="schedule-title"
                    autoComplete="off"
                    className="w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
                    value={title}
                    placeholder={t("admin.schedules.titlePlaceholder")}
                    onChange={(e) => setTitle(e.target.value)}
                  />
                </label>
                <label className="flex flex-col gap-1.5">
                  <span className="text-sm font-medium">{t("admin.schedules.descriptionLabel")}</span>
                  <textarea
                    name="schedule-description"
                    className="min-h-20 w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                </label>
              </form>
            </Modal.Body>
            <Modal.Footer>
              <Button slot="close" variant="secondary">{t("common.cancel")}</Button>
              <Button
                type="submit"
                form="create-schedule"
                isDisabled={create.isPending || title.trim().length === 0}
              >
                {t("common.create")}
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>

      <Modal.Backdrop isOpen={deleteModal.isOpen} onOpenChange={deleteModal.setOpen}>
        <Modal.Container size="sm">
          <Modal.Dialog>
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{t("admin.schedules.deleteConfirmTitle")}</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <p className="text-sm text-muted">
                {t("admin.schedules.deleteConfirmBody", { title: target?.title ?? "" })}
              </p>
            </Modal.Body>
            <Modal.Footer>
              <Button slot="close" variant="secondary">{t("common.cancel")}</Button>
              <Button
                variant="danger"
                isDisabled={remove.isPending}
                onPress={() => target && remove.mutate(target.id)}
              >
                {t("common.delete")}
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </div>
  );
}
