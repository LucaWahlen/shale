import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Card, Dropdown, Modal, toast, useOverlayState } from "@heroui/react";

import { api } from "../api/client";
import type { AdminSchedule } from "../api/client";
import { apiErrorMessage } from "../lib/errors";
import { formatDateTime, formatTimeSpan, intlLocale } from "../lib/datetime";

function DotsIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className="size-5" aria-hidden="true">
      <circle cx="5" cy="12" r="1.8" />
      <circle cx="12" cy="12" r="1.8" />
      <circle cx="19" cy="12" r="1.8" />
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

  const { data, isLoading } = useQuery({
    queryKey: ["admin-schedules"],
    queryFn: api.listSchedules,
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

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-xl font-semibold">{t("admin.schedules.title")}</h1>
          {data && data.length > 0 ? (
            <p className="text-sm text-muted" aria-live="polite">
              {t("admin.schedules.count", { count: data.length })}
            </p>
          ) : null}
        </div>
        <Button onPress={() => { resetForm(); createModal.open(); }}>{t("admin.schedules.create")}</Button>
      </div>

      {isLoading ? <p className="text-muted">{t("common.loading")}</p> : null}

      {data && data.length === 0 ? (
        <Card>
          <Card.Content className="flex flex-col items-center gap-3 py-12 text-center">
            <p className="font-medium">{t("admin.schedules.emptyTitle")}</p>
            <p className="max-w-sm text-sm text-muted">{t("admin.schedules.empty")}</p>
            <Button variant="secondary" onPress={() => { resetForm(); createModal.open(); }}>
              {t("admin.schedules.create")}
            </Button>
          </Card.Content>
        </Card>
      ) : null}

      <div className="flex flex-col gap-3">
        {data?.map((s) => {
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
            <Card key={s.id}>
              <Card.Content className="gap-2.5">
                <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
                  <div className="min-w-0 flex-1">
                    <Link
                      to={`/s/${s.id}`}
                      className="text-base font-semibold hover:underline focus-visible:ring-2 focus-visible:ring-accent"
                    >
                      {s.title}
                    </Link>
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
