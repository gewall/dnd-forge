"use client";

import { createContext, useContext, useEffect, useState } from "react";
import { createStore, type StoreApi, useStore } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { type Draft, emptyDraft } from "./draft";

export interface DraftState extends Draft {
  step: number;
  update: (patch: Partial<Draft>) => void;
  replace: (draft: Draft, step: number) => void;
  setStep: (step: number) => void;
  reset: () => void;
}

const DRAFT_KEY = "df-builder-draft";

function makeStore(initial: Draft, persistDraft: boolean): StoreApi<DraftState> {
  const init = (set: StoreApi<DraftState>["setState"]): DraftState => ({
    ...initial,
    step: 0,
    update: (patch) => set(patch),
    replace: (draft, step) => set({ ...draft, step }),
    setStep: (step) => set({ step }),
    reset: () => set({ ...emptyDraft, raceId: undefined, subraceId: undefined, classId: undefined, backgroundId: undefined, step: 0 }),
  });
  if (!persistDraft) return createStore<DraftState>()((set) => init(set));
  return createStore<DraftState>()(
    persist((set) => init(set), {
      name: DRAFT_KEY,
      version: 1,
      storage: createJSONStorage(() => localStorage),
      // Rehydrated in an effect so the server and first client render agree.
      skipHydration: true,
    }),
  );
}

const DraftContext = createContext<StoreApi<DraftState> | null>(null);

/**
 * Wizard state. A new character keeps its draft in localStorage so a refresh
 * or a detour through login does not lose it; editing starts from the saved
 * character and is not persisted.
 */
export function DraftProvider({
  initial,
  persistDraft,
  children,
}: {
  initial?: Draft;
  persistDraft: boolean;
  children: (hydrated: boolean) => React.ReactNode;
}) {
  const [store] = useState(() => makeStore(initial ?? emptyDraft, persistDraft));
  const [hydrated, setHydrated] = useState(!persistDraft);

  useEffect(() => {
    if (!persistDraft) return;
    const p = (store as unknown as { persist: { rehydrate: () => Promise<void> | void } }).persist;
    Promise.resolve(p.rehydrate()).finally(() => setHydrated(true));
  }, [store, persistDraft]);

  return <DraftContext value={store}>{children(hydrated)}</DraftContext>;
}

export function useDraft<T>(selector: (s: DraftState) => T): T {
  const store = useContext(DraftContext);
  if (!store) throw new Error("useDraft harus di dalam DraftProvider");
  return useStore(store, selector);
}
