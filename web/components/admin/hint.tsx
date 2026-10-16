import { Tooltip } from "@mantine/core";
import { ReactNode } from "react";
import styles from "./adminPanel.module.css";

interface HintProps {
  label: ReactNode;
  tip: ReactNode;
}

export default function Hint({ label, tip }: HintProps) {
  return (
    <Tooltip label={tip} multiline maw={260} withArrow openDelay={150}>
      <span className={styles.hint} tabIndex={0}>
        {label}
      </span>
    </Tooltip>
  );
}
