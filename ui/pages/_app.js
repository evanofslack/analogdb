import "../styles/globals.css";
import { BreakpointProvider } from "../providers/breakpoint";
import "@fontsource/courier-prime";

const queries = {
    xs: "(max-width: 320px)",
    sm: "(max-width: 720px)",
    md: "(max-width: 1024px)",
    lg: "(max-width: 1440px)",
};

function MyApp({ Component, pageProps }) {
    return (
        <BreakpointProvider queries={queries}>
            <Component {...pageProps} />
        </BreakpointProvider>
    );
}

export default MyApp;
