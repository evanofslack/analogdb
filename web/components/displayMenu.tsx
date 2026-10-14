"use client";

import useDisplay from "@hooks/useDisplay";
import {
  columnOptions,
  defaultDisplay,
  Gear,
  gearOptions,
  isDefault,
  Spacing,
  spacingOptions,
} from "@lib/display";
import {
  Button,
  FloatingPosition,
  Menu,
  SegmentedControl,
  VisuallyHidden,
} from "@mantine/core";
import { IconLayoutGrid } from "@tabler/icons-react";
import { useSyncExternalStore } from "react";
import styles from "./displayMenu.module.css";
import filterStyles from "./filterBar.module.css";

const touchQuery = "(hover: none)";

function subscribeTouch(listener: () => void) {
  const query = window.matchMedia(touchQuery);
  query.addEventListener("change", listener);
  return () => query.removeEventListener("change", listener);
}

interface DisplayMenuProps {
  buttonClassName?: string;
  sectionClassName?: string;
  labelClassName?: string;
  position?: FloatingPosition;
}

export default function DisplayMenu({
  buttonClassName,
  sectionClassName,
  labelClassName,
  position,
}: DisplayMenuProps) {
  const { display, width, setDisplay } = useDisplay();
  const touch = useSyncExternalStore(
    subscribeTouch,
    () => window.matchMedia(touchQuery).matches,
    () => false
  );
  const changed = !isDefault(display);

  const choices = columnOptions(width);
  const columns =
    display.columns !== "auto" && choices.includes(display.columns)
      ? String(display.columns)
      : "auto";

  // hover does nothing without a pointer, so touch screens only get off or always
  const gears = gearOptions.filter((gear) => !(touch && gear === "hover"));
  const gear = touch && display.gear === "hover" ? "off" : display.gear;

  return (
    <Menu shadow="md" width={270} position={position}>
      <Menu.Target>
        <Button
          variant="outline"
          color="gray"
          leftSection={<IconLayoutGrid size={18} stroke={1.5} />}
          classNames={{
            root: `${styles.button} ${buttonClassName ?? ""}`,
            section: sectionClassName,
          }}
          aria-label="display"
          {...(changed ? { "data-active": true } : {})}
        >
          <span className={labelClassName}>display</span>
          <span className={styles.dot} aria-hidden="true" />
          {changed && <VisuallyHidden>, customized</VisuallyHidden>}
        </Button>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Label>display</Menu.Label>
        <div className={styles.groups}>
          <div className={filterStyles.segmentGroup}>
            <h5 className={filterStyles.segmentTitle}>columns</h5>
            <SegmentedControl
              value={columns}
              onChange={(value) =>
                setDisplay({
                  columns: value === "auto" ? "auto" : Number(value),
                })
              }
              data={["auto", ...choices.map(String)]}
            />
          </div>
          <div className={filterStyles.segmentGroup}>
            <h5 className={filterStyles.segmentTitle}>spacing</h5>
            <SegmentedControl
              value={display.spacing}
              onChange={(value) => setDisplay({ spacing: value as Spacing })}
              data={[...spacingOptions]}
            />
          </div>
          <div className={filterStyles.segmentGroup}>
            <h5 className={filterStyles.segmentTitle}>gear</h5>
            <SegmentedControl
              value={gear}
              onChange={(value) => setDisplay({ gear: value as Gear })}
              data={gears}
            />
          </div>
          <button
            type="button"
            className={styles.reset}
            onClick={() => setDisplay(defaultDisplay)}
            disabled={!changed}
            aria-hidden={!changed}
          >
            reset
          </button>
        </div>
      </Menu.Dropdown>
    </Menu>
  );
}
