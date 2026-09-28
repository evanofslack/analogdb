import { Suspense } from "react";
import HomePage from "./home-page";

export const metadata = {
  title: "AnalogDB",
  description: "Film photography database",
};

export default function Page() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <HomePage />
    </Suspense>
  );
}
