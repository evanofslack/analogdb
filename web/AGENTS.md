# AGENTS.md - AnalogDB Web Development Guide

## Build/Test/Lint Commands

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run start` - Start production server
- `npm run lint` - Run ESLint (extends next/core-web-vitals)

## Code Style Guidelines

### Formatting (Prettier)

- Use double quotes, semicolons, 2-space tabs
- Trailing commas: ES5 style
- No tabs, use spaces

### Imports & Structure

- Use alias imports for lib: `@lib/client`, `@components/gallery`
- Group imports: external libraries first, then local imports
- Use named exports for components and utilities

### React/Next.js Conventions

- Use "use client" directive for client components
- Functional components with hooks (useState, useEffect, useCallback)
- CSS Modules for styling (`.module.css` files)
- Use Mantine UI components and styling patterns

### Error Handling

- Use try/catch for async operations
- Log errors with `console.error("descriptive message:", error)`
- Throw errors to propagate them up the call stack
