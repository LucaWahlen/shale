import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Card, Checkbox, Modal, toast, useOverlayState } from "@heroui/react";

import { api } from "../api/client";
import type { AdminEvent, AdminSchedule } from "../api/client";
import { formatDateTime, formatTimeSpan, todayISO } from "../lib/datetime";
import { apiErrorMessage } from "../lib/errors";
import { ShaleDatePicker, ShaleTimeField } from "../components/DateTimeFields";

type EventFormState = {
  id: string | null;
  name: string;
  description: string;
  location: string;
  date: string;
  time: string;
  endTime: string;
  allDay: boolean;
};

function compareEvents(a: AdminEvent, b: AdminEvent): number {
  return (
    a.starts_at.localeCompare(b.starts_at) ||
    a.created_at.localeCompare(b.created_at) ||
    a.id.localeCompare(b.id)
  );
}

function sortEvents(events: AdminEvent[]): AdminEvent[] {
  return [...events].sort(compareEvents);
}

function emptyEventForm(): EventFormState {
  return { id: null, name: "", description: "", location: "", date: "", time: "09:00", endTime: "", allDay: false };
}

function eventFormFromEvent(ev: AdminEvent): EventFormState {
  return {
    id: ev.id,
    name: ev.name,
    description: ev.description,
    location: ev.location,
    date: ev.starts_at.slice(0, 10),
    time: ev.starts_at.slice(11),
    endTime: ev.ends_at.slice(11),
    allDay: ev.all_day,
  };
}

function eventTimeLabel(
  ev: Pick<AdminEvent, "starts_at" | "ends_at" | "all_day">,
  locale: string,
): string {
  if (ev.all_day) return "";
  if (ev.ends_at) return formatTimeSpan(ev.starts_at, ev.ends_at, locale);
  return ev.starts_at.slice(11);
}

export function AdminScheduleEditor() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const scheduleID = id;
  const { t, i18n } = useTranslation();
  const queryClient = useQueryClient();

  const eventModal = useOverlayState();
  const deleteEventModal = useOverlayState();
  const duplicateModal = useOverlayState();
  const [form, setForm] = useState<EventFormState>(emptyEventForm());
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [attendeeNames, setAttendeeNames] = useState<Record<string, string>>({});

  const { data: schedule, isLoading } = useQuery({
    queryKey: ["admin-schedule", scheduleID],
    queryFn: () => api.getScheduleAdmin(scheduleID),
    enabled: scheduleID !== "",
  });

  useEffect(() => {
    if (schedule) {
      setTitle(schedule.title);
      setDescription(schedule.description);
    }
  }, [schedule]);

  useEffect(() => {
    setForm(emptyEventForm());
    setAttendeeNames({});
    setDeleteTarget(null);
  }, [scheduleID]);

  const invalidate = () => {
    void queryClient.invalidateQueries({
      queryKey: ["admin-schedule", scheduleID],
      refetchType: "all",
    });
    void queryClient.invalidateQueries({
      queryKey: ["admin-schedules"],
      refetchType: "all",
    });
    void queryClient.invalidateQueries({
      queryKey: ["schedule", scheduleID],
      refetchType: "all",
    });
  };

  const setScheduleCache = (updater: (current: AdminSchedule) => AdminSchedule) => {
    queryClient.setQueryData<AdminSchedule>(["admin-schedule", scheduleID], (old) =>
      old ? updater(old) : old,
    );
  };

  const dirty =
    schedule !== undefined && (title !== schedule.title || description !== schedule.description);

  useEffect(() => {
    if (!dirty) return;
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [dirty]);

  const updateSchedule = useMutation({
    mutationFn: () => api.updateSchedule(scheduleID, { title: title.trim(), description }),
    onSuccess: (updated) => {
      setScheduleCache((current) => ({
        ...current,
        title: updated.title,
        description: updated.description,
        updated_at: updated.updated_at,
      }));
      invalidate();
      toast.success(t("admin.editor.saved"));
    },
    onError: (err) => toast.danger(apiErrorMessage(err, t)),
  });

  const saveEvent = useMutation({
    mutationFn: () => {
      const startsAt = `${form.date}T${form.allDay ? "00:00" : form.time}`;
      const endsAt = form.allDay || form.endTime === "" ? "" : `${form.date}T${form.endTime}`;
      if (form.id === null) {
        return api.createEvent(scheduleID, {
          name: form.name.trim(),
          description: form.description,
          location: form.location,
          starts_at: startsAt,
          all_day: form.allDay,
          ends_at: endsAt,
        });
      }
      return api.patchEvent(form.id, {
        name: form.name.trim(),
        description: form.description,
        location: form.location,
        starts_at: startsAt,
        all_day: form.allDay,
        ends_at: endsAt,
      });
    },
    onSuccess: (saved) => {
      eventModal.close();
      setScheduleCache((current) => {
        const events = current.events ?? [];
        const exists = events.some((e) => e.id === saved.id);
        const next = exists
          ? events.map((e) => (e.id === saved.id ? { ...e, ...saved, attendees: e.attendees } : e))
          : [...events, { ...saved, attendees: [] }];
        const sorted = sortEvents(next);
        return { ...current, events: sorted, event_count: sorted.length };
      });
      invalidate();
    },
    onError: (err) => toast.danger(apiErrorMessage(err, t)),
  });

  const deleteEvent = useMutation({
    mutationFn: (eventID: string) => api.deleteEvent(eventID),
    onSuccess: (_res, eventID) => {
      deleteEventModal.close();
      setScheduleCache((current) => {
        const events = (current.events ?? []).filter((e) => e.id !== eventID);
        return { ...current, events, event_count: events.length };
      });
      invalidate();
      toast.success(t("admin.editor.eventDeleted"));
    },
    onError: () => toast.danger(t("errors.unexpected")),
  });

  const duplicate = useMutation({
    mutationFn: () => api.duplicateSchedule(scheduleID),
    onSuccess: (created) => {
      duplicateModal.close();
      void queryClient.invalidateQueries({ queryKey: ["admin-schedules"], refetchType: "all" });
      toast.success(t("admin.editor.duplicateDone"));
      navigate(`/schedules/${created.id}`);
    },
    onError: (err) => toast.danger(apiErrorMessage(err, t)),
  });

  const addAttendee = useMutation({
    mutationFn: (input: { eventID: string; name: string }) => api.addAttendee(input.eventID, input.name),
    onSuccess: (attendee, vars) => {
      setAttendeeNames((m) => ({ ...m, [vars.eventID]: "" }));
      setScheduleCache((current) => ({
        ...current,
        events: (current.events ?? []).map((e) =>
          e.id === vars.eventID ? { ...e, attendees: [...e.attendees, attendee] } : e,
        ),
      }));
      invalidate();
    },
    onError: (err) => toast.danger(apiErrorMessage(err, t)),
  });

  const removeAttendee = useMutation({
    mutationFn: (attendeeID: string) => api.removeAttendee(attendeeID),
    onSuccess: (_res, attendeeID) => {
      setScheduleCache((current) => ({
        ...current,
        events: (current.events ?? []).map((e) => ({
          ...e,
          attendees: e.attendees.filter((a) => a.id !== attendeeID),
        })),
      }));
      invalidate();
    },
    onError: () => toast.danger(t("errors.unexpected")),
  });

  if (isLoading || !schedule) {
    return <p className="text-muted">{t("common.loading")}</p>;
  }

  const shareURL = `${window.location.origin}/s/${schedule.id}`;

  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(shareURL);
      toast.success(t("admin.editor.linkCopied"));
    } catch {
      window.prompt(t("admin.editor.shareUrlLabel"), shareURL);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <Link
            to="/"
            className="inline-flex size-9 shrink-0 items-center justify-center rounded-full text-muted transition-colors hover:bg-surface hover:text-foreground focus-visible:ring-2 focus-visible:ring-accent"
            aria-label={t("admin.editor.backToList")}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="size-5" aria-hidden="true">
              <path d="m15 18-6-6 6-6" />
            </svg>
          </Link>
          <h1 className="truncate text-xl font-semibold">{schedule.title}</h1>
        </div>
        <div className="flex shrink-0 flex-wrap gap-1.5">
          <Button variant="secondary" size="sm" onPress={() => void copyLink()}>
            {t("admin.editor.copyLink")}
          </Button>
          <Button variant="secondary" size="sm" onPress={() => duplicateModal.open()}>
            {t("common.duplicate")}
          </Button>
        </div>
      </div>

      <Card>
        <Card.Content className="gap-4">
          <h2 className="text-base font-semibold">{t("admin.editor.settingsTitle")}</h2>
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              updateSchedule.mutate();
            }}
          >
            <label className="flex flex-col gap-1.5">
              <span className="text-sm font-medium">{t("admin.schedules.titleLabel")}</span>
              <input
                name="schedule-title"
                autoComplete="off"
                className="w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
                value={title}
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
            <div>
              <Button type="submit" isDisabled={updateSchedule.isPending || title.trim().length === 0}>
                {t("common.save")}
              </Button>
            </div>
          </form>
        </Card.Content>
      </Card>

      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">{t("admin.editor.eventsTitle")}</h2>
          <Button
            size="sm"
            onPress={() => {
              setForm({ ...emptyEventForm(), date: todayISO() });
              eventModal.open();
            }}
          >
            {t("admin.editor.addEvent")}
          </Button>
        </div>

        {schedule.events?.length === 0 ? <p className="text-sm text-muted">{t("schedule.noEvents")}</p> : null}

        {schedule.events?.map((ev) => {
          const timeLabel = eventTimeLabel(ev, i18n.language);
          return (
            <Card key={ev.id}>
              <Card.Content className="gap-3">
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                  <h3 className="font-semibold">{ev.name}</h3>
                  <span className="text-sm tabular-nums text-muted">
                    {formatDateTime(ev.starts_at, i18n.language)}
                    {ev.all_day ? ` · ${t("schedule.allDay")}` : timeLabel ? ` · ${timeLabel}` : ""}
                  </span>
                </div>
                {ev.description ? <p className="whitespace-pre-line text-sm text-muted">{ev.description}</p> : null}
                {ev.location ? <p className="text-sm text-muted">{ev.location}</p> : null}

                {ev.attendees.length > 0 ? (
                  <ul className="flex flex-wrap gap-1.5">
                    {ev.attendees.map((a) => (
                      <li
                        key={a.id}
                        className="flex max-w-full items-center gap-2 rounded-lg bg-surface-secondary py-1 pl-2.5 pr-1"
                      >
                        <span className="min-w-0 truncate text-sm">{a.name}</span>
                        <button
                          type="button"
                          className="grid size-6 shrink-0 place-items-center rounded-md text-muted transition-colors hover:bg-danger/10 hover:text-danger focus-visible:ring-2 focus-visible:ring-accent"
                          aria-label={`${t("common.remove")}: ${a.name}`}
                          onClick={() => removeAttendee.mutate(a.id)}
                        >
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="size-3.5" aria-hidden="true">
                            <path d="M18 6 6 18M6 6l12 12" />
                          </svg>
                        </button>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <span className="text-sm text-muted">{t("admin.editor.attendeesEmpty")}</span>
                )}

                <form
                  className="flex flex-wrap items-center gap-2"
                  onSubmit={(e) => {
                    e.preventDefault();
                    const value = (attendeeNames[ev.id] ?? "").trim();
                    if (value) addAttendee.mutate({ eventID: ev.id, name: value });
                  }}
                >
                  <input
                    name="attendee-name"
                    autoComplete="off"
                    className="min-w-0 flex-1 rounded-lg border border-border bg-surface px-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-accent"
                    placeholder={t("admin.editor.addAttendeePlaceholder")}
                    value={attendeeNames[ev.id] ?? ""}
                    onChange={(e) => setAttendeeNames((m) => ({ ...m, [ev.id]: e.target.value }))}
                    aria-label={t("admin.editor.addAttendeeLabel")}
                  />
                  <Button
                    size="sm"
                    variant="secondary"
                    type="submit"
                    isDisabled={(attendeeNames[ev.id] ?? "").trim().length === 0 || addAttendee.isPending}
                  >
                    {t("admin.editor.addAttendee")}
                  </Button>
                </form>

                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="secondary"
                    onPress={() => {
                      setForm(eventFormFromEvent(ev));
                      eventModal.open();
                    }}
                  >
                    {t("common.edit")}
                  </Button>
                  <Button
                    size="sm"
                    variant="danger"
                    onPress={() => {
                      setDeleteTarget(ev.id);
                      deleteEventModal.open();
                    }}
                  >
                    {t("common.delete")}
                  </Button>
                </div>
              </Card.Content>
            </Card>
          );
        })}
      </div>

      <Modal.Backdrop isOpen={eventModal.isOpen} onOpenChange={eventModal.setOpen}>
        <Modal.Container size="sm">
          <Modal.Dialog>
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{form.id === null ? t("admin.editor.addEvent") : t("admin.editor.editEvent")}</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <form
                id="event-form"
                className="flex flex-col gap-4"
                onSubmit={(e) => {
                  e.preventDefault();
                  saveEvent.mutate();
                }}
              >
                <label className="flex flex-col gap-1.5">
                  <span className="text-sm font-medium">{t("admin.editor.nameLabel")}</span>
                  <input
                    name="event-name"
                    autoComplete="off"
                    className="w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
                    value={form.name}
                    placeholder={t("admin.editor.namePlaceholder")}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    autoFocus
                  />
                </label>
                <label className="flex flex-col gap-1.5">
                  <span className="text-sm font-medium">{t("admin.schedules.descriptionLabel")}</span>
                  <textarea
                    name="event-description"
                    className="min-h-20 w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
                    value={form.description}
                    onChange={(e) => setForm({ ...form, description: e.target.value })}
                  />
                </label>
                <label className="flex flex-col gap-1.5">
                  <span className="text-sm font-medium">{t("admin.editor.locationLabel")}</span>
                  <input
                    name="event-location"
                    autoComplete="off"
                    className="w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus-visible:ring-2 focus-visible:ring-accent"
                    value={form.location}
                    placeholder={t("admin.editor.locationPlaceholder")}
                    onChange={(e) => setForm({ ...form, location: e.target.value })}
                  />
                </label>
                <ShaleDatePicker
                  label={t("admin.editor.dateLabel")}
                  name="event-date"
                  value={form.date}
                  onChange={(d) => setForm({ ...form, date: d })}
                />
                <Checkbox
                  name="event-all-day"
                  isSelected={form.allDay}
                  onChange={(v) => setForm({ ...form, allDay: v })}
                >
                  <Checkbox.Content>
                    <Checkbox.Control>
                      <Checkbox.Indicator />
                    </Checkbox.Control>
                    <span className="text-sm font-medium">{t("admin.editor.allDayLabel")}</span>
                  </Checkbox.Content>
                </Checkbox>
                {!form.allDay ? (
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <ShaleTimeField
                      label={t("admin.editor.timeLabel")}
                      name="event-time"
                      value={form.time}
                      onChange={(v) => setForm({ ...form, time: v })}
                    />
                    <ShaleTimeField
                      label={t("admin.editor.endTimeLabel")}
                      name="event-end-time"
                      value={form.endTime}
                      onChange={(v) => setForm({ ...form, endTime: v })}
                    />
                  </div>
                ) : null}
              </form>
            </Modal.Body>
            <Modal.Footer>
              <Button slot="close" variant="secondary">{t("common.cancel")}</Button>
              <Button
                type="submit"
                form="event-form"
                isDisabled={
                  saveEvent.isPending ||
                  form.name.trim().length === 0 ||
                  form.date === "" ||
                  (!form.allDay && form.time === "") ||
                  (!form.allDay && form.endTime !== "" && form.endTime <= form.time)
                }
              >
                {t("common.save")}
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>

      <Modal.Backdrop isOpen={deleteEventModal.isOpen} onOpenChange={deleteEventModal.setOpen}>
        <Modal.Container size="sm">
          <Modal.Dialog>
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{t("admin.editor.deleteEventConfirmTitle")}</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <p className="text-sm text-muted">
                {t("admin.editor.deleteEventConfirmBody", {
                  name: schedule.events?.find((e) => e.id === deleteTarget)?.name ?? "",
                })}
              </p>
            </Modal.Body>
            <Modal.Footer>
              <Button slot="close" variant="secondary">{t("common.cancel")}</Button>
              <Button
                variant="danger"
                isDisabled={deleteEvent.isPending}
                onPress={() => deleteTarget !== null && deleteEvent.mutate(deleteTarget)}
              >
                {t("common.delete")}
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>

      <Modal.Backdrop isOpen={duplicateModal.isOpen} onOpenChange={duplicateModal.setOpen}>
        <Modal.Container size="sm">
          <Modal.Dialog>
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{t("admin.editor.duplicate")}</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <p className="text-sm text-muted">{t("admin.editor.duplicateHint")}</p>
            </Modal.Body>
            <Modal.Footer>
              <Button slot="close" variant="secondary">{t("common.cancel")}</Button>
              <Button isDisabled={duplicate.isPending} onPress={() => duplicate.mutate()}>
                {t("common.duplicate")}
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </div>
  );
}
