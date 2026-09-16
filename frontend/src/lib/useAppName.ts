import { useQuery } from "@tanstack/react-query";

import { api } from "../api/client";

export const DEFAULT_APP_NAME = "shale";

export function useAppSettings() {
  return useQuery({
    queryKey: ["public-settings"],
    queryFn: api.getPublicSettings,
    staleTime: Infinity,
  });
}

export function useAppName(): string {
  const { data } = useAppSettings();
  return data?.app_name ?? DEFAULT_APP_NAME;
}
