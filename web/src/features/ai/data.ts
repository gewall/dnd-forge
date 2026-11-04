import "server-only";

import type { components } from "@/lib/api/schema";
import { requestApi } from "@/lib/auth/session";

export type Portrait = components["schemas"]["PortraitDTO"];
export type Lore = components["schemas"]["LoreDTO"];
export type Style = components["schemas"]["StyleDTO"];
export type LoreOptions = components["schemas"]["OptionsDTO"];
export type Quotas = components["schemas"]["QuotasDTO"];

export interface AiPanelData {
  styles: Style[];
  loreOptions: LoreOptions | null;
  portraits: Portrait[];
  lores: Lore[];
  quota: Quotas | null;
}

/** Everything the portrait and lore panels need for one character, fetched in parallel. */
export async function getAiPanelData(characterId: string): Promise<AiPanelData> {
  const client = await requestApi();
  const path = { params: { path: { id: characterId } } };
  const [styles, options, portraits, lores, quota] = await Promise.all([
    client.GET("/v1/portrait-styles"),
    client.GET("/v1/lore-options"),
    client.GET("/v1/characters/{id}/portraits", path),
    client.GET("/v1/characters/{id}/lore", path),
    client.GET("/v1/me/quota"),
  ]);
  return {
    styles: styles.data?.styles ?? [],
    loreOptions: options.data ?? null,
    portraits: portraits.data?.items ?? [],
    lores: lores.data?.items ?? [],
    quota: quota.data ?? null,
  };
}
