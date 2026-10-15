import { colorOptions } from "@lib/color";
import {
  Button,
  ButtonProps,
  Checkbox,
  Menu,
  Tooltip,
  VisuallyHidden,
} from "@mantine/core";
import { IconPalette } from "@tabler/icons-react";
import { ChangeEvent } from "react";
import styles from "./colorFilter.module.css";

interface ColorFilterProps {
  color: string[] | null;
  setColor: (color: string[] | null) => void;
  buttonClassNames?: ButtonProps["classNames"];
  labelClassName?: string;
  dotClassName?: string;
  active?: boolean;
}

export default function ColorFilter({
  color,
  setColor,
  buttonClassNames,
  labelClassName,
  dotClassName,
  active,
}: ColorFilterProps) {
  const checked = color?.length === 1 ? color[0].toLowerCase() : null;

  // a named color replaces any hexes, clicking the checked one clears
  const handleColorClick = (event: ChangeEvent<HTMLInputElement>) => {
    const clickedColor = event.target.id;
    setColor(clickedColor === checked ? null : [clickedColor]);
  };

  return (
    <Menu shadow="md" width={100}>
      <Menu.Target>
        <Button
          variant="outline"
          color="gray"
          leftSection={<IconPalette size={18} stroke={1.5} />}
          classNames={buttonClassNames}
          data-active={active || undefined}
        >
          <span className={labelClassName}>color</span>
          <span className={dotClassName} aria-hidden="true" />
          {active && <VisuallyHidden>, filter on</VisuallyHidden>}
        </Button>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Label>with color</Menu.Label>
        <div className={styles.colors}>
          {colorOptions.map((colorOption, index) => (
            <Tooltip
              key={colorOption.id}
              label={colorOption.label}
              color={
                checked === colorOption.id
                  ? colorOption.checkedColor
                  : colorOption.tooltipColor
              }
              position="right"
              withArrow
            >
              <Checkbox
                styles={{
                  input: {
                    backgroundColor: colorOption.bg,
                    border: colorOption.outline
                      ? `1px solid ${colorOption.outline}`
                      : "none",
                  },
                }}
                size="md"
                color={colorOption.colorProp}
                checked={checked === colorOption.id}
                onChange={handleColorClick}
                id={colorOption.id}
                aria-label={colorOption.label}
                className={styles.colorButton}
                radius="xs"
              />
            </Tooltip>
          ))}
        </div>
      </Menu.Dropdown>
    </Menu>
  );
}
