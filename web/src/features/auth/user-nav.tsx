import Link from "next/link";

import { Button } from "@/components/ui/button";
import { getCurrentUser } from "@/lib/auth/session";

import { UserMenu } from "./user-menu";

/** Account area of the header; reads the session, so render it inside <Suspense>. */
export async function UserNav() {
  const user = await getCurrentUser();
  if (!user) {
    return (
      <Button asChild variant="ghost">
        <Link href="/login">Masuk</Link>
      </Button>
    );
  }
  return <UserMenu user={{ name: user.name, email: user.email, isAdmin: user.role === "ADMIN" }} />;
}
