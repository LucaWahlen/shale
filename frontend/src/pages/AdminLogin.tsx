import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Button, Card, toast } from "@heroui/react";

import { api, ApiError } from "../api/client";
import { HeaderControls } from "../components/HeaderControls";
import { useAppName } from "../lib/useAppName";

export function AdminLogin() {
  const { t } = useTranslation();
  const appName = useAppName();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [password, setPassword] = useState("");

  const login = useMutation({
    mutationFn: () => api.login(password),
    onSuccess: () => {
      void queryClient.invalidateQueries();
      navigate("/", { replace: true });
    },
    onError: (err) => {
      if (err instanceof ApiError && err.code === "rate_limited") {
        toast.danger(t("errors.rate_limited"));
      } else {
        toast.danger(t("admin.login.failed"));
      }
    },
  });

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="sticky top-0 z-10 flex items-center justify-between border-b border-border/60 px-4 py-3 sm:px-6">
        <Link to="/" className="text-lg font-semibold tracking-tight focus-visible:ring-2 focus-visible:ring-accent">
          {appName}
        </Link>
        <HeaderControls />
      </header>
      <main className="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center px-4 py-12">
        <Card>
          <Card.Content className="gap-4 p-6">
            <h1 className="text-xl font-bold">{t("admin.login.title")}</h1>
            <form
              className="flex flex-col gap-4"
              onSubmit={(e) => {
                e.preventDefault();
                login.mutate();
              }}
            >
              <label className="flex flex-col gap-1.5">
                <span className="text-sm font-medium">{t("admin.login.passwordLabel")}</span>
                <input
                  type="password"
                  className="w-full rounded-lg border border-border bg-surface px-3 py-2.5 text-base outline-none focus:ring-2 focus:ring-accent"
                  value={password}
                  placeholder={t("admin.login.passwordPlaceholder")}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  autoFocus
                />
              </label>
              <Button type="submit" isDisabled={login.isPending || password.length === 0}>
                {t("admin.login.submit")}
              </Button>
            </form>
          </Card.Content>
        </Card>
      </main>
    </div>
  );
}
