import { useMemo, useState } from "react";
import { Link, useParams } from "react-router";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Card, Chip, Modal, toast, useOverlayState } from "@heroui/react";

import { api, ApiError } from "../api/client";
import { HeaderControls } from "../components/HeaderControls";
import { apiErrorMessage } from "../lib/errors";
import { formatDate, formatTimeSpan, splitStartsAt } from "../lib/datetime";
import { useAppName } from "../lib/useAppName";
import {
  getAttendanceEntry,
  getProfileName,
  recordAttendance,
  removeAttendance,
  setProfileName,
} from "../lib/identity";
import type { PublicEvent, PublicSchedule } from "../api/client";
import type { ReactNode } from "react";

function EventCard({ event, scheduleID }: { event: PublicEvent; scheduleID: string }) {
  const { t, i18n } = useTranslation();
  const queryClient = useQueryClient();
  const modal = useOverlayState();
  const revertModal = useOverlayState();
  const [name, setName] = useState("");

  const mine = getAttendanceEntry(event.id);
  const timeLabel = event.all_day
    ? t("schedule.allDay")
    : event.ends_at
      ? formatTimeSpan(event.starts_at, event.ends_at, i18n.language)
      : splitStartsAt(event.starts_at).time;

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["schedule", scheduleID] });

  const attendMutation = useMutation({
    mutationFn: async (attendeeName: string) => api.attend(scheduleID, event.id, attendeeName),
    onMutate: async (attendeeName: string) => {
      await queryClient.cancelQueries({ queryKey: ["schedule", scheduleID] });
      const previous = queryClient.getQueryData<PublicSchedule>(["schedule", scheduleID]);
      queryClient.setQueryData<PublicSchedule>(["schedule", scheduleID], (old) => {
        if (!old) return old;
        return {
          ...old,
          events: old.events.map((e) =>
            e.id === event.id && !e.attendees.some((a) => a.name.toLowerCase() === attendeeName.trim().toLowerCase())
              ? { ...e, attendees: [...e.attendees, { id: `optimistic-${Date.now()}-${Math.floor(Math.random() * 1e6)}`, name: attendeeName.trim() }] }
              : e,
          ),
        };
      });
      return { previous };
    },
    onError: (err, _name, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["schedule", scheduleID], context.previous);
      }
      toast.danger(apiErrorMessage(err, t));
    },
    onSuccess: (res) => {
      recordAttendance(event.id, {
        attendeeId: res.attendee.id,
        name: res.attendee.name,
        token: res.manage_token,
        slug: String(scheduleID),
      });
      setProfileName(res.attendee.name);
      setName("");
      modal.close();
      void invalidate();
    },
  });

  const revertMutation = useMutation({
    mutationFn: (entry: { attendeeId: string; token: string }) =>
      api.revert(scheduleID, event.id, entry.attendeeId, entry.token),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ["schedule", scheduleID] });
      const previous = queryClient.getQueryData(["schedule", scheduleID]);
      queryClient.setQueryData(["schedule", scheduleID], (old: PublicSchedule | undefined) => {
        if (!old) return old;
        return {
          ...old,
          events: old.events.map((e) =>
            e.id === event.id ? { ...e, attendees: e.attendees.filter((a) => a.id !== mine?.attendeeId) } : e,
          ),
        };
      });
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["schedule", scheduleID], context.previous);
      }
      removeAttendance(event.id);
      void queryClient.invalidateQueries({ queryKey: ["schedule", scheduleID] });
      revertModal.close();
    },
    onSuccess: () => {
      removeAttendance(event.id);
      revertModal.close();
      void invalidate();
    },
  });

  const submitAttend = () => {
    const trimmed = name.trim();
    if (!trimmed) {
      toast.warning(t("schedule.failedName"));
      return;
    }
    attendMutation.mutate(trimmed);
  };

  const attendeeCount = event.attendees.length;
  const dimmed = !event.attendable;

  return (
    <Card className={dimmed ? "opacity-60" : undefined}>
      <Card.Content className="gap-3">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <h3 className="text-lg font-semibold">{event.name}</h3>
          <span className="text-sm tabular-nums text-muted">{timeLabel}</span>
        </div>
        {event.description ? <p className="whitespace-pre-line text-sm text-muted">{event.description}</p> : null}
        <div className="flex flex-wrap items-center gap-2 text-sm text-muted">
          {event.location ? <span>{event.location}</span> : null}
          {!event.attendable ? (
            <Chip variant="soft" size="sm">
              {t("schedule.closed")}
            </Chip>
          ) : null}
        </div>

        {attendeeCount > 0 ? (
          <ul className="flex flex-wrap gap-1.5">
            {event.attendees.map((a) => {
              const isMine = mine !== undefined && a.name.toLowerCase() === mine.name.toLowerCase();
              return (
                <li
                  key={a.id}
                  className={`flex max-w-full items-center gap-1.5 rounded-lg bg-surface-secondary py-1 pl-2.5 pr-2.5 ${
                    isMine ? "ring-1 ring-accent/60" : ""
                  }`}
                >
                  <span className="min-w-0 truncate text-sm">{a.name}</span>
                  {isMine ? <span className="shrink-0 text-xs text-muted">{t("schedule.you")}</span> : null}
                </li>
              );
            })}
          </ul>
        ) : (
          <span className="text-sm text-muted">{t("schedule.attendeesEmpty")}</span>
        )}

        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-sm text-muted">{t("schedule.attendees", { count: attendeeCount })}</span>
          <div className="flex gap-2">
            {mine ? (
              <>
                <span className="inline-flex items-center text-sm font-medium text-success">
                  {t("schedule.attending")}
                </span>
                {event.attendable ? (
                  <Button size="sm" variant="secondary" onPress={revertModal.open}>
                    {t("schedule.revert")}
                  </Button>
                ) : null}
              </>
            ) : event.attendable ? (
              <Button
                size="sm"
                onPress={() => {
                  setName(getProfileName());
                  modal.open();
                }}
              >
                {t("schedule.attend")}
              </Button>
            ) : null}
          </div>
        </div>

        <Modal.Backdrop isOpen={modal.isOpen} onOpenChange={modal.setOpen}>
          <Modal.Container size="sm">
            <Modal.Dialog>
              <Modal.CloseTrigger />
              <Modal.Header>
                <Modal.Heading>{t("schedule.attendTitle")}</Modal.Heading>
              </Modal.Header>
              <Modal.Body>
                <div className="flex flex-col gap-4">
                  <label className="flex flex-col gap-1.5">
                    <span className="text-sm font-medium">{t("schedule.nameLabel")}</span>
                    <input
                      name="attendee-name"
                      autoComplete="off"
                      className="w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
                      value={name}
                      placeholder={t("schedule.namePlaceholder")}
                      onChange={(e) => setName(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") submitAttend();
                      }}
                     />
                   </label>
                 </div>
               </Modal.Body>
              <Modal.Footer>
                <Button slot="close" variant="secondary">
                  {t("common.cancel")}
                </Button>
                <Button onPress={submitAttend} isDisabled={attendMutation.isPending}>
                  {t("schedule.attendConfirm")}
                </Button>
              </Modal.Footer>
            </Modal.Dialog>
          </Modal.Container>
        </Modal.Backdrop>

        <Modal.Backdrop isOpen={revertModal.isOpen} onOpenChange={revertModal.setOpen}>
          <Modal.Container size="sm">
            <Modal.Dialog>
              <Modal.CloseTrigger />
              <Modal.Header>
                <Modal.Heading>{t("schedule.revertTitle")}</Modal.Heading>
              </Modal.Header>
              <Modal.Body>
                <p className="text-sm text-muted">{t("schedule.revertHint")}</p>
              </Modal.Body>
              <Modal.Footer>
                <Button slot="close" variant="secondary">
                  {t("common.cancel")}
                </Button>
                <Button
                  variant="danger"
                  isDisabled={revertMutation.isPending}
                  onPress={() => mine && revertMutation.mutate({ attendeeId: mine.attendeeId, token: mine.token })}
                >
                  {t("schedule.revertConfirm")}
                </Button>
              </Modal.Footer>
            </Modal.Dialog>
          </Modal.Container>
        </Modal.Backdrop>
      </Card.Content>
    </Card>
  );
}

export function SchedulePage() {
  const { id = "" } = useParams();
  const scheduleID = id;
  const { t, i18n } = useTranslation();
  const appName = useAppName();
  const { data, isError, error } = useQuery({
    queryKey: ["schedule", scheduleID],
    queryFn: () => api.getSchedule(scheduleID),
    enabled: scheduleID !== "",
    retry: false,
  });

  const eventGroups = useMemo(() => (data ? groupScheduleEvents(data) : []), [data]);

  if (isError) {
    const notFound = error instanceof ApiError && error.status === 404;
    return (
      <Shell>
        <h1 className="text-2xl font-bold">{t("schedule.notFound")}</h1>
        <p className="text-muted">{notFound ? t("schedule.notFoundHint") : t("errors.network")}</p>
      </Shell>
    );
  }

  if (!data) {
    return (
      <Shell>
        <p className="text-muted">{t("common.loading")}</p>
      </Shell>
    );
  }

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="sticky top-0 z-10 flex items-center justify-between border-b border-border/60 bg-background/80 px-4 py-3 backdrop-blur sm:px-6">
        <Link to="/" className="text-lg font-semibold tracking-tight focus-visible:ring-2 focus-visible:ring-accent">
          {appName}
        </Link>
        <HeaderControls />
      </header>
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 py-6 sm:px-6">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-balance sm:text-3xl">{data.title}</h1>
          {data.description ? <p className="mt-2 whitespace-pre-line text-muted">{data.description}</p> : null}
          <p className="mt-2 text-sm text-muted">{t("schedule.eventsCount", { count: data.events.length })}</p>
        </div>
        {eventGroups.length === 0 ? (
          <p className="text-muted">{t("schedule.noEvents")}</p>
        ) : (
          <div className="flex flex-col gap-8">
            {eventGroups.map((g) => (
              <section key={g.date} className="flex flex-col gap-3">
                <h2 className="text-sm font-semibold uppercase tracking-wide text-muted">
                  {formatDate(g.date, i18n.language)}
                </h2>
                {g.events.map((ev) => (
                  <EventCard key={ev.id} event={ev} scheduleID={scheduleID} />
                ))}
              </section>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}

function groupScheduleEvents(data: PublicSchedule): { date: string; events: PublicEvent[] }[] {
  const groups: { date: string; events: PublicEvent[] }[] = [];
  for (const ev of data.events) {
    const date = ev.starts_at.slice(0, 10);
    const last = groups[groups.length - 1];
    if (last && last.date === date) {
      last.events.push(ev);
    } else {
      groups.push({ date, events: [ev] });
    }
  }
  return groups;
}

function Shell({ children }: { children: ReactNode }) {
  const appName = useAppName();
  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="sticky top-0 z-10 flex items-center justify-between border-b border-border/60 bg-background/80 px-4 py-3 backdrop-blur sm:px-6">
        <Link to="/" className="text-lg font-semibold tracking-tight focus-visible:ring-2 focus-visible:ring-accent">
          {appName}
        </Link>
        <HeaderControls />
      </header>
      <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-4 px-4 py-12 sm:px-6">{children}</main>
    </div>
  );
}
