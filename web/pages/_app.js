import "../styles/globals.css";
import { BreakpointProvider } from "../providers/breakpoint";
import "@fontsource/courier-prime";
import { MantineProvider } from "@mantine/core";
import { NavigationProgress } from "@mantine/nprogress";

const queries = {
  xs: "(max-width: 360px)",
  sm: "(max-width: 720px)",
  md: "(max-width: 1024px)",
  lg: "(max-width: 1440px)",
};

function MyApp({ Component, pageProps }) {
  return (
    <MantineProvider>
      <NavigationProgress />
      <BreakpointProvider queries={queries}>
        <Component {...pageProps} />
      </BreakpointProvider>
    </MantineProvider>
  );
}

export default MyApp;
