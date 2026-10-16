"use client";

import { statsParsers } from "@lib/stats";
import { Switch } from "@mantine/core";
import { useQueryStates } from "nuqs";

export default function NsfwSwitch() {
  const [{ nsfw }, setParams] = useQueryStates(statsParsers);
  return (
    <Switch
      size="xs"
      color="gray"
      label="Show NSFW"
      checked={nsfw === "include"}
      onChange={(event) =>
        setParams({ nsfw: event.currentTarget.checked ? "include" : null })
      }
    />
  );
}
