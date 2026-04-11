"use client";

import dynamic from "next/dynamic";
import { SimpleGrid, Text, Title } from "@mantine/core";
import styles from "./stats.module.css";
import type {
  ServerStatsCamerasResponse,
  ServerStatsColorsResponse,
  ServerStatsFilmsResponse,
  ServerStatsOverviewResponse,
  ServerStatsPeriodsResponse,
} from "analogdb-generated";

const ResponsiveContainer = dynamic(
  () => import("recharts").then((m) => m.ResponsiveContainer),
  { ssr: false }
);
const LineChart = dynamic(
  () => import("recharts").then((m) => m.LineChart),
  { ssr: false }
);
const BarChart = dynamic(
  () => import("recharts").then((m) => m.BarChart),
  { ssr: false }
);
const Line = dynamic(() => import("recharts").then((m) => m.Line), {
  ssr: false,
});
const Bar = dynamic(() => import("recharts").then((m) => m.Bar), {
  ssr: false,
});
const Cell = dynamic(() => import("recharts").then((m) => m.Cell), {
  ssr: false,
});
const XAxis = dynamic(() => import("recharts").then((m) => m.XAxis), {
  ssr: false,
});
const YAxis = dynamic(() => import("recharts").then((m) => m.YAxis), {
  ssr: false,
});
const CartesianGrid = dynamic(
  () => import("recharts").then((m) => m.CartesianGrid),
  { ssr: false }
);
const Tooltip = dynamic(() => import("recharts").then((m) => m.Tooltip), {
  ssr: false,
});
const Legend = dynamic(() => import("recharts").then((m) => m.Legend), {
  ssr: false,
});

interface StatsProps {
  overview: ServerStatsOverviewResponse;
  overTime: ServerStatsPeriodsResponse;
  films: ServerStatsFilmsResponse;
  cameras: ServerStatsCamerasResponse;
  colors: ServerStatsColorsResponse;
}

function formatNumber(n?: number): string {
  if (n == null) return "—";
  return n.toLocaleString();
}

function formatScore(n?: number): string {
  if (n == null) return "—";
  return n.toLocaleString(undefined, { maximumFractionDigits: 0 });
}

function unixToYear(ts?: number): string {
  if (ts == null) return "—";
  return new Date(ts * 1000).getFullYear().toString();
}

export default function StatsPage({
  overview,
  overTime,
  films,
  cameras,
  colors,
}: StatsProps) {
  const d = overview.data;

  const overTimeData = (overTime.data ?? []).map((p) => ({
    period: p.period?.slice(0, 7) ?? "",
    count: p.count ?? 0,
    avgScore: p.avgScore ?? 0,
  }));

  const filmsData = (films.data ?? []).map((f) => ({
    name: `${f.filmMake ?? ""} ${f.filmType ?? ""}`.trim(),
    postCount: f.postCount ?? 0,
    avgScore: Math.round(f.avgScore ?? 0),
  }));

  const camerasData = (cameras.data ?? []).map((c) => ({
    name: `${c.cameraMake ?? ""} ${c.cameraModel ?? ""}`.trim(),
    postCount: c.postCount ?? 0,
    avgScore: Math.round(c.avgScore ?? 0),
  }));

  const colorsData = (colors.data ?? []).map((c) => ({
    name: c.htmlName ?? "",
    hex: c.hex ?? "#888",
    postCount: c.postCount ?? 0,
  }));

  return (
    <div className={styles.page}>
      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Overview
        </Title>
        <SimpleGrid cols={{ base: 2, sm: 3 }} spacing="md">
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>total posts</Text>
            <Text className={styles.tileValue}>{formatNumber(d?.totalPosts)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>avg score</Text>
            <Text className={styles.tileValue}>{formatScore(d?.avgScore)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>cameras</Text>
            <Text className={styles.tileValue}>{formatNumber(d?.totalCameras)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>film stocks</Text>
            <Text className={styles.tileValue}>{formatNumber(d?.totalFilms)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>earliest post</Text>
            <Text className={styles.tileValue}>{unixToYear(d?.earliestPost)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>latest post</Text>
            <Text className={styles.tileValue}>{unixToYear(d?.latestPost)}</Text>
          </div>
        </SimpleGrid>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Posts Over Time
        </Title>
        <div className={styles.chartWrap}>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={overTimeData} margin={{ top: 5, right: 30, left: 10, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
              <XAxis
                dataKey="period"
                tick={{ fontSize: 11 }}
                interval="preserveStartEnd"
              />
              <YAxis yAxisId="left" tick={{ fontSize: 11 }} />
              <YAxis
                yAxisId="right"
                orientation="right"
                tick={{ fontSize: 11 }}
              />
              <Tooltip />
              <Legend />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="count"
                stroke="#1a1a1a"
                dot={false}
                strokeWidth={2}
              />
              <Line
                yAxisId="right"
                type="monotone"
                dataKey="avgScore"
                stroke="#888"
                dot={false}
                strokeWidth={1.5}
                strokeDasharray="5 5"
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Top Film Stocks
        </Title>
        <div className={styles.chartWrap}>
          <ResponsiveContainer width="100%" height={Math.max(300, filmsData.length * 26)}>
            <BarChart
              layout="vertical"
              data={filmsData}
              margin={{ top: 5, right: 30, left: 10, bottom: 5 }}
            >
              <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="#e0e0e0" />
              <XAxis type="number" tick={{ fontSize: 11 }} />
              <YAxis
                type="category"
                dataKey="name"
                width={130}
                tick={{ fontSize: 11 }}
              />
              <Tooltip />
              <Bar dataKey="postCount" fill="#1a1a1a" radius={[0, 2, 2, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Top Cameras
        </Title>
        <div className={styles.chartWrap}>
          <ResponsiveContainer width="100%" height={Math.max(300, camerasData.length * 26)}>
            <BarChart
              layout="vertical"
              data={camerasData}
              margin={{ top: 5, right: 30, left: 10, bottom: 5 }}
            >
              <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="#e0e0e0" />
              <XAxis type="number" tick={{ fontSize: 11 }} />
              <YAxis
                type="category"
                dataKey="name"
                width={130}
                tick={{ fontSize: 11 }}
              />
              <Tooltip />
              <Bar dataKey="postCount" fill="#1a1a1a" radius={[0, 2, 2, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Top Colors
        </Title>
        <div className={styles.chartWrap}>
          <ResponsiveContainer width="100%" height={Math.max(300, colorsData.length * 22)}>
            <BarChart
              layout="vertical"
              data={colorsData}
              margin={{ top: 5, right: 30, left: 10, bottom: 5 }}
            >
              <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="#e0e0e0" />
              <XAxis type="number" tick={{ fontSize: 11 }} />
              <YAxis
                type="category"
                dataKey="name"
                width={100}
                tick={{ fontSize: 11 }}
              />
              <Tooltip />
              <Bar dataKey="postCount" radius={[0, 2, 2, 0]}>
                {colorsData.map((entry, i) => (
                  <Cell key={i} fill={entry.hex} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      </section>
    </div>
  );
}
