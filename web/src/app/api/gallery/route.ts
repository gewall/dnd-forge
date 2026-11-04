import type { NextRequest } from "next/server";

import { jsonReply } from "@/features/ai/route-helpers";
import { getGalleryPage } from "@/features/gallery/data";

/** Next gallery page for "Muat lagi"; served from the same cache as the page. */
export async function GET(request: NextRequest) {
  const before = request.nextUrl.searchParams.get("before") ?? undefined;
  if (before && Number.isNaN(Date.parse(before))) {
    return jsonReply({ status: 422, detail: "cursor tidak valid" }, 422);
  }
  return jsonReply(await getGalleryPage(before), 200);
}
