"use client";

import { ActionIcon, useMantineColorScheme } from "@mantine/core";
import { IconMoon, IconSun } from "@tabler/icons-react";

export default function ThemeToggle({ size = 22, stroke = 1.5 }) {
  const { toggleColorScheme } = useMantineColorScheme();

  // render both icons and let mantine css hide one so ssr markup never depends on scheme
  return (
    <ActionIcon
      onClick={toggleColorScheme}
      variant="subtle"
      color="gray"
      c="var(--adb-heading)"
      size="lg"
      aria-label="Toggle color scheme"
    >
      <IconMoon size={size} stroke={stroke} className="mantine-dark-hidden" />
      <IconSun size={size} stroke={stroke} className="mantine-light-hidden" />
    </ActionIcon>
  );
}
