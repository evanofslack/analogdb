"use client";

import useIsAdmin from "@hooks/useIsAdmin";
import { ActionIcon, Menu } from "@mantine/core";
import { useDisclosure, useMediaQuery } from "@mantine/hooks";
import { IconCheck, IconMenu2 } from "@tabler/icons-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect } from "react";
import { FiCode, FiGithub } from "react-icons/fi";
import styles from "./mobileNav.module.css";
import ThemeToggle from "./themeToggle";

interface NavLink {
  href: string;
  label: string;
  active: (path: string) => boolean;
}

const LINKS: NavLink[] = [
  { href: "/", label: "Gallery", active: (path) => path === "/" },
  {
    href: "/search",
    label: "Search",
    active: (path) => path.startsWith("/search"),
  },
  {
    href: "/films",
    label: "Film",
    active: (path) => path.startsWith("/films"),
  },
  {
    href: "/cameras",
    label: "Cameras",
    active: (path) => path.startsWith("/cameras"),
  },
  {
    href: "/stats",
    label: "Stats",
    active: (path) => path.startsWith("/stats"),
  },
  { href: "/about", label: "About", active: (path) => path === "/about" },
];

const ADMIN_LINK: NavLink = {
  href: "/admin",
  label: "Admin",
  active: (path) => path === "/admin",
};

interface MobileNavProps {
  onOpenChange?: (opened: boolean) => void;
}

export default function MobileNav({ onOpenChange }: MobileNavProps) {
  const isAdmin = useIsAdmin();
  const pathname = usePathname();
  const [opened, { close, toggle }] = useDisclosure(false);
  const isDesktop = useMediaQuery("(min-width: 721px)");

  useEffect(() => {
    if (isDesktop) {
      close();
    }
  }, [isDesktop, close]);

  useEffect(() => {
    onOpenChange?.(opened);
  }, [opened, onOpenChange]);

  const links = isAdmin ? [...LINKS, ADMIN_LINK] : LINKS;
  const docsActive = pathname === "/docs";

  return (
    <div className={styles.bar}>
      <ThemeToggle size={24} stroke={2} />
      <Menu
        opened={opened}
        onClose={close}
        position="bottom-end"
        offset={6}
        width={220}
        radius="md"
        shadow="md"
        transitionProps={{ transition: "pop-top-right", duration: 150 }}
        classNames={{
          dropdown: styles.dropdown,
          item: styles.item,
          divider: styles.divider,
        }}
      >
        <Menu.Target>
          <ActionIcon
            onClick={toggle}
            variant="subtle"
            color="gray"
            c="var(--adb-heading)"
            size="xl"
            aria-label="Menu"
          >
            <IconMenu2 size={24} stroke={2} />
          </ActionIcon>
        </Menu.Target>
        <Menu.Dropdown>
          {links.map((link) => {
            const isActive = link.active(pathname);
            return (
              <Menu.Item
                key={link.href}
                component={Link}
                href={link.href}
                aria-current={isActive ? "page" : undefined}
                className={isActive ? styles.active : undefined}
              >
                <span className={styles.label}>
                  {link.label}
                  {isActive && (
                    <IconCheck size={18} stroke={2} aria-hidden="true" />
                  )}
                </span>
              </Menu.Item>
            );
          })}
          <Menu.Divider />
          <Menu.Item
            component={Link}
            href="/docs"
            aria-current={docsActive ? "page" : undefined}
            className={`${styles.github} ${docsActive ? styles.active : ""}`}
          >
            <span className={styles.label}>
              <FiCode size="1rem" aria-hidden="true" />
              API
              {docsActive && (
                <IconCheck size={18} stroke={2} aria-hidden="true" />
              )}
            </span>
          </Menu.Item>
          <Menu.Item
            component="a"
            href="https://github.com/evanofslack/analogdb"
            className={styles.github}
          >
            <span className={styles.label}>
              <FiGithub size="1rem" aria-hidden="true" />
              GitHub
            </span>
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>
    </div>
  );
}
