import { isUuid, jsonReply } from "@/features/ai/route-helpers";
import { requestApi } from "@/lib/auth/session";

/** Status and text of a lore draft for the panel's polling. */
export async function GET(_request: Request, { params }: RouteContext<"/api/lore/[id]">) {
  const { id } = await params;
  if (!isUuid(id)) return jsonReply({ status: 404, detail: "lore tidak ditemukan" }, 404);

  const { data, error, response } = await (await requestApi()).GET("/v1/lore/{id}", { params: { path: { id } } });
  return jsonReply(data ?? error, response.status);
}
