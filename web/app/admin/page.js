import AdminPanel, { adminTabs } from "@components/admin/adminPanel";
import AuditTab from "@components/admin/auditTab";
import LoginForm from "@components/admin/loginForm";
import OverviewTab from "@components/admin/overviewTab";
import QualityTab from "@components/admin/qualityTab";
import ReportsTab from "@components/admin/reportsTab";
import ReviewTab from "@components/admin/reviewTab";
import TabError from "@components/admin/tabError";
import TrafficTab from "@components/admin/trafficTab";
import {
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

export const metadata = {
  title: "Admin",
  robots: { index: false, follow: false },
};

function positiveInt(value) {
  const n = Number(value);
  return Number.isInteger(n) && n > 0 ? n : null;
}

function pick(value, allowed, fallback) {
  return allowed.includes(value) ? value : fallback;
}

async function load(fn) {
  try {
    return { data: await fn() };
  } catch (error) {
    console.error("admin load failed:", error);
    return { error: { status: error?.status ?? 500, message: error?.message } };
  }
}

async function renderTab(tab, params) {
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
    case "traffic": {
      const range = pick(params.range, trafficRanges, "7d");
      const { data, error } = await load(() => getTraffic(range));
      return <TrafficTab traffic={data} error={error} range={range} />;
    }
    case "audit": {
      const before = positiveInt(params.before);
      const { data, error } = await load(() => getAudit(before));
      if (error) return <TabError error={error} />;
      return <AuditTab page={data} />;
    }
    default: {
      const { data, error } = await load(getOverview);
      if (error) return <TabError error={error} />;
      return <OverviewTab overview={data} now={Date.now()} />;
    }
  }
}

function externalLinks() {
  const links = [];
  if (process.env.GRAFANA_URL) {
    links.push({ label: "Grafana", href: process.env.GRAFANA_URL });
  }
  if (process.env.DAGSTER_URL) {
    links.push({ label: "Dagster", href: process.env.DAGSTER_URL });
  }
  return links;
}

export default async function AdminPage({ searchParams }) {
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
