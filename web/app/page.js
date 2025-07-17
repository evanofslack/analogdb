import { checkAdminAuth } from "@lib/auth";
import { Suspense } from "react";
import HomePage from "./home-page";

export const metadata = {
  title: "AnalogDB",
  description: "Film photography database",
};

export default async function Page() {
  const isAdmin = await checkAdminAuth();
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <HomePage isAdmin={isAdmin} />
    </Suspense>
  );
}
