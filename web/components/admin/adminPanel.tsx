import { logoutAction } from "@app/actions/auth";
import Header from "@components/header";
import { Button, Title } from "@mantine/core";
import { IconLogout } from "@tabler/icons-react";
import Link from "next/link";
import { ReactNode } from "react";
import styles from "./adminPanel.module.css";

export interface AdminLink {
  label: string;
  href: string;
}

interface AdminPanelProps {
  tab: string;
  links: AdminLink[];
  children: ReactNode;
}

export const adminTabs = [
  { id: "overview", label: "Overview" },
  { id: "review", label: "Review" },
  { id: "reports", label: "Reports" },
  { id: "quality", label: "Quality" },
  { id: "analytics", label: "Analytics" },
  { id: "traffic", label: "Traffic" },
  { id: "audit", label: "Audit" },
];

export default function AdminPanel({ tab, links, children }: AdminPanelProps) {
  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.container}>
        <div className={styles.header}>
          <Title order={1} size="h2">
            Admin
          </Title>
          <div className={styles.headerLinks}>
            {links.map((link) => (
              <a
                key={link.label}
                href={link.href}
                target="_blank"
                rel="noreferrer"
                className={styles.externalLink}
              >
                {link.label} ↗
              </a>
            ))}
            <form action={logoutAction}>
              <Button
                type="submit"
                color="red"
                variant="outline"
                size="xs"
                leftSection={<IconLogout size={14} />}
              >
                Logout
              </Button>
            </form>
          </div>
        </div>

        <nav className={styles.tabs}>
          {adminTabs.map((t) => (
            <Link
              key={t.id}
              href={`/admin?tab=${t.id}`}
              className={`${styles.tab} ${
                t.id === tab ? styles.tabActive : ""
              }`}
              aria-current={t.id === tab ? "page" : undefined}
            >
              {t.label}
            </Link>
          ))}
        </nav>

        {children}
      </div>
    </div>
  );
}
