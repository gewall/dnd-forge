"use client";

import { useEffect, useRef } from "react";

import { type JobStatus, POLL_INTERVAL_MS, pollStep } from "./poll";

/**
 * Polls `url` every 2 s while `active`, pausing while the tab is hidden, and
 * reports each response. Stops once the job leaves PENDING or after ~3 min.
 */
export function useJobPoll<T extends { status: JobStatus }>(
  url: string | null,
  handlers: { onUpdate: (item: T) => void; onTimeout: () => void; onError: (message: string) => void },
) {
  const latest = useRef(handlers);
  useEffect(() => {
    latest.current = handlers;
  });

  useEffect(() => {
    if (!url) return;
    let tries = 0;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    const schedule = () => {
      if (!stopped) timer = setTimeout(tick, POLL_INTERVAL_MS);
    };

    const tick = async () => {
      if (stopped) return;
      if (document.hidden) {
        // Resume on visibilitychange instead of burning tries in the background.
        return;
      }
      tries += 1;
      try {
        const res = await fetch(url, { cache: "no-store" });
        const body = await res.json().catch(() => null);
        if (stopped) return;
        if (!res.ok) {
          stopped = true;
          latest.current.onError(body?.detail ?? "Status tidak bisa dimuat.");
          return;
        }
        const item = body as T;
        latest.current.onUpdate(item);
        const step = pollStep(item, tries);
        if (step.done) return;
        if (step.timedOut) {
          stopped = true;
          latest.current.onTimeout();
          return;
        }
      } catch {
        // Network blip: try again on the next tick.
      }
      schedule();
    };

    const onVisible = () => {
      if (!document.hidden && !stopped) {
        clearTimeout(timer);
        tick();
      }
    };

    document.addEventListener("visibilitychange", onVisible);
    schedule();
    return () => {
      stopped = true;
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [url]);
}
