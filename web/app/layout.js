import "@styles/globals.css";
import "@mantine/core/styles.css";
import "@mantine/charts/styles.css";
import "@mantine/code-highlight/styles.css";
import "@mantine/dropzone/styles.css";
import Analytics from "@components/analytics";
import {
  ColorSchemeScript,
  mantineHtmlProps,
  MantineProvider,
} from "@mantine/core";
import { CodeHighlightProvider } from "@providers/codehighlight";
import { QueryProvider } from "@providers/query";
import { theme } from "@providers/theme";
import { NuqsAdapter } from "nuqs/adapters/next/app";

export const metadata = {
  metadataBase: new URL("https://analogdb.com"),
  title: { default: "AnalogDB", template: "%s | AnalogDB" },
  description: "The collection of film photography",
  openGraph: {
    siteName: "AnalogDB",
    type: "website",
  },
  twitter: { card: "summary_large_image" },
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" {...mantineHtmlProps}>
      <head>
        <ColorSchemeScript defaultColorScheme="light" />
        <script
          defer
          src="https://umami.eslack.net/script.js"
          data-website-id="dac2f8b3-f9d3-4089-b091-8a2221cd4871"
          data-do-not-track="true"
        />
      </head>
      <body>
        <Analytics />
        <MantineProvider theme={theme} defaultColorScheme="light">
          <CodeHighlightProvider>
            <NuqsAdapter>
              <QueryProvider>{children}</QueryProvider>
            </NuqsAdapter>
          </CodeHighlightProvider>
        </MantineProvider>
      </body>
    </html>
  );
}
