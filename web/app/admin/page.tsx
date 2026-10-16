import AdminPanel, { AdminLink, adminTabs } from "@components/admin/adminPanel";
import AnalyticsTab from "@components/admin/analyticsTab";
import AuditTab from "@components/admin/auditTab";
import LoginForm from "@components/admin/loginForm";
import OverviewTab from "@components/admin/overviewTab";
import QualityTab from "@components/admin/qualityTab";
import ReportsTab from "@components/admin/reportsTab";
import ReviewTab from "@components/admin/reviewTab";
import TabError, { LoadError } from "@components/admin/tabError";
import TrafficTab from "@components/admin/trafficTab";
import {
  getAnalytics,
  getAudit,
  getMissingPosts,
  getOverview,
  getQuality,
  getReports,
  getReviewPosts,
  getTraffic,
  missingFields,
  reportStatuses,
  trafficRanges,
} from "@lib/adminClient";
import { checkAdminAuth } from "@lib/auth";
import { Metadata } from "next";

type AdminSearchParams = Record<string, string | string[] | undefined>;

interface AdminPageProps {
  searchParams: Promise<AdminSearchParams>;
}

export const metadata: Metadata = {
  title: "Admin",
  robots: { index: false, follow: false },
};

function positiveInt(value: unknown): number | null {
  const n = Number(value);
  return Number.isInteger(n) && n > 0 ? n : null;
}

function pick<T extends string>(
  value: unknown,
  allowed: readonly T[],
  fallback: T
): T {
  return allowed.includes(value as T) ? (value as T) : fallback;
}

async function load<T>(
  fn: () => Promise<T>
): Promise<{ data?: T; error?: LoadError }> {
  try {
    return { data: await fn() };
  } catch (error) {
    console.error("admin load failed:", error);
    return { error: { status: error?.status ?? 500, message: error?.message } };
  }
}

async function renderTab(tab: string, params: AdminSearchParams) {
  switch (tab) {
    case "review": {
      const cursor = typeof params.cursor === "string" ? params.cursor : null;
      const { data, error } = await load(() => getReviewPosts(cursor));
      if (error) return <TabError error={error} />;
      return <ReviewTab page={data} firstPage={!cursor} />;
    }
    case "reports": {
      const status = pick(params.status, reportStatuses, "open");
      const before = positiveInt(params.before);
      const { data, error } = await load(() => getReports(status, before));
      if (error) return <TabError error={error} />;
      return <ReportsTab page={data} status={status} firstPage={!before} />;
    }
    case "quality": {
      const field = pick(params.field, missingFields, "camera");
      const beforeId = positiveInt(params.before_id);
      const [quality, missing] = await Promise.all([
        load(getQuality),
        load(() => getMissingPosts(field, beforeId)),
      ]);
      if (quality.error) return <TabError error={quality.error} />;
      return (
        <QualityTab
          quality={quality.data}
          missing={missing.data}
          missingError={missing.error}
          field={field}
        />
      );
    }
    case "analytics": {
      const range = pick(params.range, trafficRanges, "7d");
      const { data, error } = await load(() => getAnalytics(range));
      return <AnalyticsTab analytics={data} error={error} range={range} />;
    }
    case "traffic": {
      const range = pick(params.range, trafficRanges, "7d");
      const { data, error } = await load(() => getTraffic(range));
      return <TrafficTab traffic={data} error={error} range={range} />;
    }
    case "audit": {
      const before = positiveInt(params.before);
      const { data, error } = await load(() => getAudit(before));
      if (error) return <TabError error={error} />;
      return <AuditTab page={data} firstPage={!before} />;
    }
    default: {
      const { data, error } = await load(getOverview);
      if (error) return <TabError error={error} />;
      return <OverviewTab overview={data} now={Date.now()} />;
    }
  }
}

function externalLinks() {
  const links: AdminLink[] = [];
  if (process.env.GRAFANA_URL) {
    links.push({ label: "Grafana", href: process.env.GRAFANA_URL });
  }
  if (process.env.DAGSTER_URL) {
    links.push({ label: "Dagster", href: process.env.DAGSTER_URL });
  }
  return links;
}

export default async function AdminPage({ searchParams }: AdminPageProps) {
  const params = (await searchParams) ?? {};
  const isAdmin = await checkAdminAuth();

  if (!isAdmin) {
    return <LoginForm error={params.error} />;
  }

  const tab = pick(
    params.tab,
    adminTabs.map((t) => t.id),
    "overview"
  );

  return (
    <AdminPanel tab={tab} links={externalLinks()}>
      {await renderTab(tab, params)}
    </AdminPanel>
  );
}
