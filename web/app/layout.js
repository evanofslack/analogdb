import "@styles/globals.css";
import "@mantine/core/styles.css";
import "@mantine/code-highlight/styles.css";
import {
  ColorSchemeScript,
  mantineHtmlProps,
  MantineProvider,
} from "@mantine/core";
import { BreakpointProvider } from "@providers/breakpoint";
import { CodeHighlightProvider } from "@providers/codehighlight";
import { NuqsAdapter } from "nuqs/adapters/next/app";

export const metadata = {
  title: "AnalogDB",
  description: "The collection of film photography",
};

const queries = {
  xs: "(max-width: 480px)",
  sm: "(max-width: 720px)",
  md: "(max-width: 1024px)",
  lg: "(max-width: 1440px)",
  xl: "(max-width: 2048px)",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" {...mantineHtmlProps}>
      <head>
        <ColorSchemeScript />
        <script
          defer
          src="https://umami.eslack.net/script.js"
          data-website-id="dac2f8b3-f9d3-4089-b091-8a2221cd4871"
          data-do-not-track="true"
        />
      </head>
      <body>
        <MantineProvider>
          <CodeHighlightProvider>
            <NuqsAdapter>
              <BreakpointProvider queries={queries}>
                {children}
              </BreakpointProvider>
            </NuqsAdapter>
          </CodeHighlightProvider>
        </MantineProvider>
      </body>
    </html>
  );
}
