import { useEffect, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";
import { ApiError } from "./client";

export type Resource<T> = { status: "loading" } | { status: "ready"; data: T } | { status: "error"; error: unknown };
export function useResource<T>(load: (signal: AbortSignal) => Promise<T>) {
  const [resource, setResource] = useState<Resource<T>>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);
  const { expire } = useAuth();
  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal).then(data => {
      if (!controller.signal.aborted) setResource({ status: "ready", data });
    }).catch((error: unknown) => {
      if (controller.signal.aborted) return;
      if (error instanceof ApiError && error.status === 401) expire();
      else setResource({ status: "error", error });
    });
    return () => controller.abort();
  }, [load, attempt, expire]);
  return {
    resource,
    refresh: () => { setResource({ status: "loading" }); setAttempt(value => value + 1); },
    refreshQuietly: () => setAttempt(value => value + 1),
  };
}
