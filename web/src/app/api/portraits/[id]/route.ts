import { revalidateTag } from "next/cache";

import { isUuid, jsonReply } from "@/features/ai/route-helpers";
import { galleryTag } from "@/features/characters/tags";
import { requestApi } from "@/lib/auth/session";

/** Status of a portrait job for the panel's polling; the session cookie authorizes it. */
export async function GET(_request: Request, { params }: RouteContext<"/api/portraits/[id]">) {
  const { id } = await params;
  if (!isUuid(id)) return jsonReply({ status: 404, detail: "portrait tidak ditemukan" }, 404);

  const { data, error, response } = await (await requestApi()).GET("/v1/portraits/{id}", {
    params: { path: { id } },
  });
  // A new portrait may change a public character's gallery card.
  if (data?.status === "SUCCEEDED") revalidateTag(galleryTag, "max");
  return jsonReply(data ?? error, response.status);
}
