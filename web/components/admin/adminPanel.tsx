import { logoutAction } from "@app/actions/auth";
import Header from "@components/header";
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
  counts?: Record<string, number>;
  children: ReactNode;
}

export const adminTabs = [
  { id: "overview", label: "Overview" },
  { id: "moderation", label: "Moderation" },
  { id: "catalog", label: "Catalog" },
  { id: "analytics", label: "Analytics" },
];

export const tabIds = [...adminTabs.map((t) => t.id), "activity"];

export const legacyTabs: Record<string, string> = {
  review: "moderation",
  reports: "moderation",
  quality: "catalog",
  traffic: "analytics",
  audit: "activity",
};

export default function AdminPanel({
  tab,
  links,
  counts,
  children,
}: AdminPanelProps) {
  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.title}>Admin</h1>
          <div className={styles.headerLinks}>
            <Link
              href="/admin?tab=activity"
              className={styles.textLink}
              aria-current={tab === "activity" ? "page" : undefined}
            >
              Activity
            </Link>
            {links.map((link) => (
              <a
                key={link.label}
                href={link.href}
                target="_blank"
                rel="noreferrer"
                className={styles.textLink}
              >
                {link.label} ↗
              </a>
            ))}
            <form action={logoutAction}>
              <button type="submit" className={styles.textButton}>
                <IconLogout size={14} stroke={1.5} />
                Logout
              </button>
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
              {counts?.[t.id] ? (
                <span className={styles.tabCount}>{counts[t.id]}</span>
              ) : null}
            </Link>
          ))}
        </nav>

        {children}
      </div>
    </div>
  );
}
