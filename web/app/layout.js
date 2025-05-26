import '../styles/globals.css'
import '@mantine/core/styles.css';
import '@mantine/code-highlight/styles.css';
import { ColorSchemeScript, MantineProvider, mantineHtmlProps } from '@mantine/core';
import { NuqsAdapter } from 'nuqs/adapters/next/app'
import { BreakpointProvider } from "../providers/breakpoint";
import { NavigationProgress } from "@mantine/nprogress";
import { CodeHighlightProvider } from '../components/CodeHighlightProvider';

export const metadata = {
    title: 'AnalogDB',
    description: 'The collection of film photography',
}

const queries = {
    xs: "(max-width: 480px)",
    sm: "(max-width: 720px)",
    md: "(max-width: 1024px)",
    lg: "(max-width: 1440px)",
    xl: "(max-width: 2048px)",
};


export default function RootLayout({ children }) {
    return (
        <html lang="en" {...mantineHtmlProps} >
            <head>
                <ColorSchemeScript />
            </head>
            <body>
                <MantineProvider>
                    <CodeHighlightProvider>
                        <NuqsAdapter>
                            <NavigationProgress />
                            <BreakpointProvider queries={queries}>
                                {children}
                            </BreakpointProvider>
                        </NuqsAdapter>
                    </CodeHighlightProvider>
                </MantineProvider>
            </body>
        </html>
    )
}
