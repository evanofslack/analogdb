"use client";
import Gallery from "@components/gallery";
import { BreakpointProvider } from "@providers/breakpoint";

const queries = {
  xs: "(max-width: 480px)",
  sm: "(max-width: 720px)",
  md: "(max-width: 1024px)",
  lg: "(max-width: 1440px)",
  xl: "(max-width: 2048px)",
};

export default function HomePage({ data, isAdmin }) {
  return (
    <BreakpointProvider queries={queries}>
      <Gallery data={data} isAdmin={isAdmin} />
    </BreakpointProvider>
  );
}
