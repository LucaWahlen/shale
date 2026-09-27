

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
  is_past: boolean;
  events?: AdminEvent[];
  created_at: string;
  updated_at: string;
}

export type ScheduleSort = "newest" | "oldest" | "title" | "title_desc" | "updated" | "soonest" | "events";

export interface ScheduleListParams {
  page?: number;
  pageSize?: number;
  sort?: ScheduleSort;
  q?: string;
  includePast?: boolean;
}

export interface AdminSchedulePage {
  items: AdminSchedule[];
  total: number;
  page: number;
  page_size: number;
}

export interface AppSettings {
  app_name: string;
  default_language: "en" | "de";
  imprint_text: string;
  privacy_text: string;
  audit_retention_days?: number;
}

export interface AuditEntry {
  id: string;
  created_at: string;
  action: string;
  actor: string;
  actor_name: string;
  schedule_id: string;
  schedule_title: string;
  event_id: string;
  event_name: string;
  detail: string;
}

export interface AuditListParams {
  page?: number;
  pageSize?: number;
  action?: string;
  scheduleId?: string;
  q?: string;
}

export interface AuditPage {
  items: AuditEntry[];
  total: number;
  page: number;
  page_size: number;
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

  listSchedules: (params: ScheduleListParams = {}) => {
    const query = new URLSearchParams();
    if (params.page !== undefined) query.set("page", String(params.page));
    if (params.pageSize !== undefined) query.set("page_size", String(params.pageSize));
    if (params.sort !== undefined) query.set("sort", params.sort);
    if (params.q) query.set("q", params.q);
    if (params.includePast) query.set("include_past", "1");
    const qs = query.toString();
    return request<AdminSchedulePage>("GET", `/api/v1/admin/schedules${qs ? `?${qs}` : ""}`);
  },
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

  listAudit: (params: AuditListParams = {}) => {
    const query = new URLSearchParams();
    if (params.page !== undefined) query.set("page", String(params.page));
    if (params.pageSize !== undefined) query.set("page_size", String(params.pageSize));
    if (params.action) query.set("action", params.action);
    if (params.scheduleId) query.set("schedule_id", params.scheduleId);
    if (params.q) query.set("q", params.q);
    const qs = query.toString();
    return request<AuditPage>("GET", `/api/v1/admin/audit${qs ? `?${qs}` : ""}`);
  },

  exportUrl: "/api/v1/admin/export" as const,
  import: (raw: string) =>
    request<ImportResult>("POST", "/api/v1/admin/import", JSON.parse(raw)),
};
