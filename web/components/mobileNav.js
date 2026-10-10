"use client";

import useIsAdmin from "@hooks/useIsAdmin";
import { ActionIcon, Drawer } from "@mantine/core";
import { useDisclosure, useMediaQuery } from "@mantine/hooks";
import { IconCheck, IconMenu2 } from "@tabler/icons-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect } from "react";
import { FiGithub } from "react-icons/fi";
import styles from "./mobileNav.module.css";
import ThemeToggle from "./themeToggle";

const LINKS = [
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
  { href: "/about", label: "About", active: (path) => path === "/about" },
  { href: "/docs", label: "API", active: (path) => path === "/docs" },
];

const ADMIN_LINK = {
  href: "/admin",
  label: "Admin",
  active: (path) => path === "/admin",
};

export default function MobileNav() {
  const isAdmin = useIsAdmin();
  const pathname = usePathname();
  const [opened, { open, close }] = useDisclosure(false);
  const isDesktop = useMediaQuery("(min-width: 721px)");

  useEffect(() => {
    if (isDesktop) {
      close();
    }
  }, [isDesktop, close]);

  const links = isAdmin ? [...LINKS, ADMIN_LINK] : LINKS;

  return (
    <div className={styles.bar}>
      <ThemeToggle size={24} stroke={2} />
      <ActionIcon
        onClick={open}
        variant="subtle"
        color="gray"
        c="var(--adb-heading)"
        size="xl"
        aria-label="Open menu"
        aria-expanded={opened}
        aria-controls="mobile-menu"
      >
        <IconMenu2 size={24} stroke={2} />
      </ActionIcon>
      <Drawer
        id="mobile-menu"
        opened={opened}
        onClose={close}
        position="right"
        size="100%"
        title="AnalogDB"
        closeButtonProps={{ "aria-label": "Close menu", size: "xl" }}
        classNames={{
          header: styles.header,
          title: styles.title,
          content: styles.content,
          body: styles.body,
        }}
      >
        <nav className={styles.nav}>
          {links.map((link) => {
            const isActive = link.active(pathname);
            return (
              <Link
                key={link.href}
                href={link.href}
                onClick={close}
                aria-current={isActive ? "page" : undefined}
                className={styles.link}
              >
                <span className={styles.label}>{link.label}</span>
                {isActive && (
                  <IconCheck size={28} stroke={1.5} aria-hidden="true" />
                )}
              </Link>
            );
          })}
        </nav>
        <div className={styles.footer}>
          <p> &copy; 2026 AnalogDB </p>
          <a href="https://github.com/evanofslack/analogdb" aria-label="GitHub">
            <FiGithub size="1.2rem" />
          </a>
        </div>
      </Drawer>
    </div>
  );
}
