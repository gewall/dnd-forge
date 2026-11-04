export type RouteArea = "user" | "admin" | "guest";

const userPrefixes = ["/characters", "/profile"];
const guestPaths = ["/login", "/register"];

function matches(pathname: string, prefix: string): boolean {
  return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

/**
 * Which access rule proxy applies to a path: "user" and "admin" need a
 * session, "guest" pages (login, register) send signed-in users home.
 */
export function routeArea(pathname: string): RouteArea | null {
  if (matches(pathname, "/admin")) return "admin";
  if (userPrefixes.some((p) => matches(pathname, p))) return "user";
  if (guestPaths.some((p) => matches(pathname, p))) return "guest";
  return null;
}
