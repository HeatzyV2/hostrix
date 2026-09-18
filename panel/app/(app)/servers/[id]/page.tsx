import { ServerDetail } from "@/components/server-detail";
import { getMe } from "@/lib/auth";
import { redirect } from "next/navigation";

export default async function ServerDetailPage() {
  const user = await getMe();
  if (!user) redirect("/login");
  return (
    <div className="mx-auto max-w-5xl">
      <ServerDetail isAdmin={user.is_admin} />
    </div>
  );
}
