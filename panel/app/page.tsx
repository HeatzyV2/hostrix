import { redirect } from "next/navigation";
import { getMe } from "@/lib/auth";

export default async function HomePage() {
  const me = await getMe();
  redirect(me ? "/dashboard" : "/login");
}
