"use client";

import {
  CodeHighlightAdapterProvider,
  createShikiAdapter,
} from "@mantine/code-highlight";
import { ReactNode } from "react";

// Shiki requires async code to load the highlighter
async function loadShiki() {
  const { createHighlighter } = await import("shiki");
  const shiki = await createHighlighter({
    langs: ["tsx", "scss", "html", "bash", "json"],
    themes: [],
  });
  return shiki;
}

const shikiAdapter = createShikiAdapter(loadShiki);

interface CodeHighlightProviderProps {
  children: ReactNode;
}

export function CodeHighlightProvider({
  children,
}: CodeHighlightProviderProps) {
  return (
    <CodeHighlightAdapterProvider adapter={shikiAdapter}>
      {children}
    </CodeHighlightAdapterProvider>
  );
}
