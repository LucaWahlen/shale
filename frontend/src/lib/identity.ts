export interface AttendanceEntry {
  attendeeId: string;
  name: string;
  token: string;
  slug: string;
}

const PROFILE_KEY = "shale.profile";
const ATTENDANCE_KEY = "shale.attendance";

export function getProfileName(): string {
  try {
    const raw = localStorage.getItem(PROFILE_KEY);
    if (!raw) return "";
    const parsed = JSON.parse(raw) as { name?: string };
    return typeof parsed.name === "string" ? parsed.name : "";
  } catch {
    return "";
  }
}

export function setProfileName(name: string): void {
  localStorage.setItem(PROFILE_KEY, JSON.stringify({ name }));
}

function readAttendance(): Record<string, AttendanceEntry> {
  try {
    const raw = localStorage.getItem(ATTENDANCE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, AttendanceEntry>;
    }
    return {};
  } catch {
    return {};
  }
}

function writeAttendance(map: Record<string, AttendanceEntry>): void {
  localStorage.setItem(ATTENDANCE_KEY, JSON.stringify(map));
}

export function getAttendanceEntry(eventID: string): AttendanceEntry | undefined {
  return readAttendance()[String(eventID)];
}

export function recordAttendance(eventID: string, entry: AttendanceEntry): void {
  const map = readAttendance();
  map[String(eventID)] = entry;
  writeAttendance(map);
}

export function removeAttendance(eventID: string): void {
  const map = readAttendance();
  delete map[String(eventID)];
  writeAttendance(map);
}
