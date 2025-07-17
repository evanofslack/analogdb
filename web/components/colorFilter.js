import { Button, Checkbox, Menu, Tooltip } from "@mantine/core";
import { IconPalette } from "@tabler/icons-react";
import styles from "./colorFilter.module.css";

export default function ColorFilter({ color, setColor, onlyIcon }) {
  const handleColorClick = (event) => {
    let clickedColor = event.target.id;
    if (clickedColor === color) {
      setColor(null);
    } else {
      setColor(clickedColor);
    }
  };

  const colorOptions = [
    {
      id: "red",
      label: "red",
      bg: "#f03e3e",
      tooltipColor: "red.7",
      checkedColor: "red.8",
      colorProp: "red.8",
    },
    {
      id: "orange",
      label: "orange",
      bg: "#fd7e14",
      tooltipColor: "orange.6",
      checkedColor: "orange.7",
      colorProp: "orange.7",
    },
    {
      id: "tan",
      label: "beige",
      bg: "#ffdcb0",
      tooltipColor: "brown.1",
      checkedColor: "brown.2",
      colorProp: "brown.2",
    },
    {
      id: "yellow",
      label: "yellow",
      bg: "#ffd43b",
      tooltipColor: "yellow.4",
      checkedColor: "yellow.5",
      colorProp: "yellow.5",
    },
    {
      id: "green",
      label: "green",
      bg: "#2f9e44",
      tooltipColor: "green.8",
      checkedColor: "green.9",
      colorProp: "green.9",
    },
    {
      id: "olive",
      label: "olive",
      bg: "#4c4d00",
      tooltipColor: "olive.8",
      checkedColor: "olive.9",
      colorProp: "olive.9",
    },
    {
      id: "teal",
      label: "teal",
      bg: "#22b8cf",
      tooltipColor: "cyan.5",
      checkedColor: "cyan.6",
      colorProp: "cyan.6",
    },
    {
      id: "navy",
      label: "navy",
      bg: "#064679",
      tooltipColor: "navy.7",
      checkedColor: "navy.8",
      colorProp: "navy.8",
    },
    {
      id: "purple",
      label: "purple",
      bg: "#9c36b5",
      tooltipColor: "grape.8",
      checkedColor: "grape.9",
      colorProp: "grape.9",
    },
    {
      id: "gray",
      label: "gray",
      bg: "#868e96",
      tooltipColor: "dark.2",
      checkedColor: "dark.3",
      colorProp: "dark.3",
    },
    {
      id: "brown",
      label: "brown",
      bg: "#7d4500",
      tooltipColor: "brown.7",
      checkedColor: "brown.8",
      colorProp: "brown.8",
    },
    {
      id: "black",
      label: "black",
      bg: "#141517",
      tooltipColor: "dark.8",
      checkedColor: "dark.9",
      colorProp: "dark.9",
    },
    {
      id: "white",
      label: "white",
      bg: "#e9ecef",
      tooltipColor: "gray.2",
      checkedColor: "gray.3",
      colorProp: "gray.3",
    },
  ];

  return (
    <Menu shadow="md" width={100}>
      <Menu.Target>
        <Button
          variant="outline"
          color="gray"
          leftSection={<IconPalette size={onlyIcon ? 22 : 18} stroke={1.5} />}
          styles={() => ({
            root: {
              marginRight: 10,
              paddingLeft: 10,
              paddingRight: onlyIcon ? 0 : 10,
              color: "#2E2E2E",
              fontWeight: 400,
              borderColor: "#CED4DA",
              "&:hover": {
                backgroundColor: "#fbfbfc",
              },
              leftSection: {
                marginRight: 5,
              },
            },
          })}
        >
          {!onlyIcon && <span>color</span>}
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
                color === colorOption.id
                  ? colorOption.checkedColor
                  : colorOption.tooltipColor
              }
              position="right"
              withArrow
            >
              <Checkbox
                styles={{
                  input: { backgroundColor: colorOption.bg, border: "None" },
                }}
                size="md"
                color={colorOption.colorProp}
                checked={color === colorOption.id}
                onChange={handleColorClick}
                id={colorOption.id}
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
