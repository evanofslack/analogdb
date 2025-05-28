'use client';

import { CodeHighlightAdapterProvider, createShikiAdapter } from '@mantine/code-highlight';

// Shiki requires async code to load the highlighter
async function loadShiki() {
    const { createHighlighter } = await import('shiki');
    const shiki = await createHighlighter({
        langs: ['tsx', 'scss', 'html', 'bash', 'json'],
        themes: [],
    });
    return shiki;
}

const shikiAdapter = createShikiAdapter(loadShiki);

export function CodeHighlightProvider({ children }) {
    return (
        <CodeHighlightAdapterProvider adapter={shikiAdapter}>
            {children}
        </CodeHighlightAdapterProvider>
    );
}
