import "../styles/globals.css";
import { BreakpointProvider } from "../providers/breakpoint";
import "@fontsource/courier-prime";
import { MantineProvider } from "@mantine/core";
import { NavigationProgress } from "@mantine/nprogress";

const queries = {
  xs: "(max-width: 480px)",
  sm: "(max-width: 720px)",
  md: "(max-width: 1024px)",
  lg: "(max-width: 1440px)",
  xl: "(max-width: 2048px)",
};

function MyApp({ Component, pageProps }) {
  return (
    <MantineProvider
      theme={{
        colors: {
          brown: [
            "#fff2dd",
            "#ffdcb0",
            "#fec481",
            "#fcad50",
            "#fb9620",
            "#e17c08",
            "#af6003",
            "#7d4500",
            "#4c2800",
            "#1e0c00",
          ],

          olive: [
            "#ffffdd",
            "#ffffb0",
            "#ffff80",
            "#ffff4f",
            "#ffff23",
            "#e5e611",
            "#b2b306",
            "#7f8000",
            "#4c4d00",
            "#191a00",
          ],
          navy: [
            "#e0f4ff",
            "#b3dcfe",
            "#86c5f9",
            "#58aef5",
            "#2f97f1",
            "#1b7dd7",
            "#1061a9",
            "#064679",
            "#002a4b",
            "#000f1d",
          ],
        },
      }}
    >
      <NavigationProgress />
      <BreakpointProvider queries={queries}>
        <Component {...pageProps} />
      </BreakpointProvider>
    </MantineProvider>
  );
}

export default MyApp;
