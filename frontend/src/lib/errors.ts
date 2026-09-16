

import { ApiError } from "../api/client";

const LOCALIZED_CODES = new Set([
  "invalid",
  "not_found",
  "conflict",
  "unauthorized",
  "forbidden",
  "not_attendable",
  "rate_limited",
  "unprocessable",
]);

export function apiErrorMessage(err: unknown, t: (key: string) => string): string {
  if (err instanceof ApiError && LOCALIZED_CODES.has(err.code)) {
    return t(`errors.${err.code}`);
  }
  if (err instanceof TypeError) {
    return t("errors.network");
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return t("errors.unexpected");
}
