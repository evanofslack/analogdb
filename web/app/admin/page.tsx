import ActivityTab, { activityViews } from "@components/admin/activityTab";
import AdminPanel, {
  AdminLink,
  legacyTabs,
  tabIds,
} from "@components/admin/adminPanel";
import AnalyticsTab from "@components/admin/analyticsTab";
import CatalogTab from "@components/admin/catalogTab";
import LoginForm from "@components/admin/loginForm";
import ModerationTab from "@components/admin/moderationTab";
import OverviewTab from "@components/admin/overviewTab";
import TabError, { LoadError } from "@components/admin/tabError";
import {
  AdminQueue,
  getAnalytics,
  getAudit,
  getMissingPosts,
  getOverview,
  getQuality,
  getQueue,
  getReports,
  getReviewPosts,
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

type QueueLoad = Promise<{ data?: AdminQueue; error?: LoadError }>;

async function renderTab(
  tab: string,
  params: AdminSearchParams,
  queueLoad: QueueLoad
) {
  switch (tab) {
    case "moderation": {
      const status = pick(params.status, reportStatuses, "open");
      const before = positiveInt(params.before);
      const cursor = typeof params.cursor === "string" ? params.cursor : null;
      const [reports, incoming, queue] = await Promise.all([
        load(() => getReports(status, before)),
        load(() => getReviewPosts(cursor)),
        queueLoad,
      ]);
      return (
        <ModerationTab
          params={{ status, before, cursor }}
          reports={reports.data}
          reportsError={reports.error}
          incoming={incoming.data}
          incomingError={incoming.error}
          openReports={queue.data?.open_reports}
        />
      );
    }
    case "catalog": {
      const field = pick(params.field, missingFields, "camera");
      const beforeId = positiveInt(params.before_id);
      const [quality, missing, queue] = await Promise.all([
        load(getQuality),
        load(() => getMissingPosts(field, beforeId)),
        queueLoad,
      ]);
      if (quality.error) return <TabError error={quality.error} />;
      return (
        <CatalogTab
          quality={quality.data}
          queue={queue.data}
          missing={missing.data}
          missingError={missing.error}
          field={field}
          firstPage={!beforeId}
        />
      );
    }
    case "analytics": {
      const range = pick(params.range, trafficRanges, "7d");
      const { data, error } = await load(() => getAnalytics(range));
      return (
        <AnalyticsTab
          analytics={data}
          error={error}
          range={range}
          grafana={process.env.GRAFANA_URL}
        />
      );
    }
    case "activity": {
      const view = pick(params.view, activityViews, "admin");
      const before = positiveInt(params.before);
      const { data, error } = await load(() =>
        getAudit(before, view === "admin" ? "web" : null)
      );
      if (error) return <TabError error={error} />;
      return <ActivityTab page={data} view={view} before={before} />;
    }
    default: {
      const [{ data, error }, queue] = await Promise.all([
        load(getOverview),
        queueLoad,
      ]);
      if (error) return <TabError error={error} />;
      return (
        <OverviewTab
          overview={data}
          queue={queue.data}
          web={{
            version: process.env.APP_VERSION || "dev",
            uptime_seconds: Math.floor(process.uptime()),
          }}
          now={Date.now()}
        />
      );
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

function tabCounts(queue?: AdminQueue) {
  if (!queue) return {};
  return {
    moderation: queue.open_reports,
    catalog: queue.unmatched_cameras + queue.unmatched_films,
  };
}

export default async function AdminPage({ searchParams }: AdminPageProps) {
  const params = (await searchParams) ?? {};
  const isAdmin = await checkAdminAuth();

  if (!isAdmin) {
    return <LoginForm error={params.error} />;
  }

  const requested =
    typeof params.tab === "string"
      ? legacyTabs[params.tab] ?? params.tab
      : params.tab;
  const tab = pick(requested, tabIds, "overview");

  const queueLoad = load(getQueue);
  const [queue, content] = await Promise.all([
    queueLoad,
    renderTab(tab, params, queueLoad),
  ]);

  return (
    <AdminPanel
      tab={tab}
      links={externalLinks()}
      counts={tabCounts(queue.data)}
    >
      {content}
    </AdminPanel>
  );
}
