"use client";

import { ActionIcon } from "@mantine/core";
import { useSearchModal } from "@providers/search";
import { IconSearch } from "@tabler/icons-react";

export default function SearchButton({ size = 22 }) {
  const { open } = useSearchModal();
  return (
    <ActionIcon
      onClick={() => open()}
      variant="subtle"
      color="gray"
      size="lg"
      aria-label="search"
    >
      <IconSearch size={size} stroke={1.5} />
    </ActionIcon>
  );
}
