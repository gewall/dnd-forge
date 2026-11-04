export const POLL_INTERVAL_MS = 2000;
/** ~3 minutes at 2 s: the API documents that most jobs finish well within that. */
export const POLL_MAX_TRIES = 90;

export type JobStatus = "PENDING" | "SUCCEEDED" | "FAILED";

export type PollStep<T> = { done: true; item: T } | { done: false; timedOut: boolean };

/** Decides what a poll tick means: finished, keep waiting, or give up. */
export function pollStep<T extends { status: JobStatus }>(item: T, tries: number): PollStep<T> {
  if (item.status !== "PENDING") return { done: true, item };
  return { done: false, timedOut: tries >= POLL_MAX_TRIES };
}

/** Replaces an item in a newest-first list by id, or puts it at the front. */
export function upsertById<T extends { id: string }>(list: T[], item: T): T[] {
  const i = list.findIndex((x) => x.id === item.id);
  if (i === -1) return [item, ...list];
  const next = list.slice();
  next[i] = item;
  return next;
}
