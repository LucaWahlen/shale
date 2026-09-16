

export interface PublicAttendee {
  id: string;
  name: string;
}

export interface PublicEvent {
  id: string;
  name: string;
  description: string;
  location: string;
  starts_at: string;
  all_day: boolean;
  ends_at: string;
  attendable: boolean;
  attendees: PublicAttendee[];
}

export interface PublicSchedule {
  id: string;
  title: string;
  description: string;
  events: PublicEvent[];
}

export interface AttendResponse {
  attendee: PublicAttendee;
  manage_token: string;
  created: boolean;
}

export interface AdminAttendee {
  id: string;
  name: string;
}

export interface AdminEvent {
  id: string;
  schedule_id: string;
  name: string;
  description: string;
  location: string;
  starts_at: string;
  all_day: boolean;
  ends_at: string;
  attendees: AdminAttendee[];
  created_at: string;
  updated_at: string;
}

export interface AdminSchedule {
  id: string;
  title: string;
  description: string;
  event_count: number;
  first_starts_at?: string;
  last_starts_at?: string;
  events?: AdminEvent[];
  created_at: string;
  updated_at: string;
}

export interface AppSettings {
  app_name: string;
  default_language: "en" | "de";
}

export interface ImportCounts {
  schedules: number;
  events: number;
  attendees: number;
}

export interface ImportResult {
  imported: ImportCounts;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details: { path: string; message: string }[];

  constructor(status: number, code: string, message: string, details: { path: string; message: string }[] = []) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    credentials: "same-origin",
  });
  if (!res.ok) {
    let code = "error";
    let message = `HTTP ${res.status}`;
    let details: { path: string; message: string }[] = [];
    try {
      const parsed = (await res.json()) as { error?: { code?: string; message?: string; details?: typeof details } };
      if (parsed.error) {
        code = parsed.error.code ?? code;
        message = parsed.error.message ?? message;
        details = parsed.error.details ?? details;
      }
    } catch {
    }
    throw new ApiError(res.status, code, message, details);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export const api = {
  getPublicSettings: () => request<AppSettings>("GET", "/api/v1/settings"),

  getSchedule: (id: string) => request<PublicSchedule>("GET", `/api/v1/schedules/${id}`),

  attend: (scheduleID: string, eventID: string, name: string) =>
    request<AttendResponse>(
      "POST",
      `/api/v1/schedules/${scheduleID}/events/${eventID}/attend`,
      { name },
    ),

  revert: (scheduleID: string, eventID: string, attendeeID: string, manageToken: string) =>
    request<void>(
      "DELETE",
      `/api/v1/schedules/${scheduleID}/events/${eventID}/attend`,
      { attendee_id: attendeeID, manage_token: manageToken },
    ),

  login: (password: string) => request<{ status: string }>("POST", "/api/v1/admin/login", { password }),
  logout: () => request<void>("POST", "/api/v1/admin/logout"),

  listSchedules: () => request<AdminSchedule[]>("GET", "/api/v1/admin/schedules"),
  createSchedule: (input: { title: string; description?: string }) =>
    request<AdminSchedule>("POST", "/api/v1/admin/schedules", input),
  getScheduleAdmin: (id: string) => request<AdminSchedule>("GET", `/api/v1/admin/schedules/${id}`),
  updateSchedule: (id: string, input: { title: string; description: string }) =>
    request<AdminSchedule>("PUT", `/api/v1/admin/schedules/${id}`, input),
  deleteSchedule: (id: string) => request<void>("DELETE", `/api/v1/admin/schedules/${id}`),
  duplicateSchedule: (id: string, title?: string) =>
    request<AdminSchedule>(
      "POST",
      `/api/v1/admin/schedules/${id}/duplicate`,
      title?.trim() ? { title: title.trim() } : undefined,
    ),

  createEvent: (
    scheduleID: string,
    input: { name: string; description?: string; location?: string; starts_at: string; all_day?: boolean; ends_at?: string },
  ) => request<AdminEvent>("POST", `/api/v1/admin/schedules/${scheduleID}/events`, input),
  patchEvent: (
    eventID: string,
    patch: { name?: string; description?: string; location?: string; starts_at?: string; all_day?: boolean; ends_at?: string },
  ) => request<AdminEvent>("PATCH", `/api/v1/admin/events/${eventID}`, patch),
  deleteEvent: (eventID: string) => request<void>("DELETE", `/api/v1/admin/events/${eventID}`),

  addAttendee: (eventID: string, name: string) =>
    request<AdminAttendee>("POST", `/api/v1/admin/events/${eventID}/attendees`, { name }),
  removeAttendee: (attendeeID: string) =>
    request<void>("DELETE", `/api/v1/admin/attendees/${attendeeID}`),

  getAdminSettings: () => request<AppSettings>("GET", "/api/v1/admin/settings"),
  putAdminSettings: (input: AppSettings) =>
    request<AppSettings>("PUT", "/api/v1/admin/settings", input),

  exportUrl: "/api/v1/admin/export" as const,
  import: (raw: string) =>
    request<ImportResult>("POST", "/api/v1/admin/import", JSON.parse(raw)),
};
