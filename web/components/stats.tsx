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
const Line = dynamic(() => import("recharts").then((m) => m.Line), { ssr: false });
const Bar = dynamic(() => import("recharts").then((m) => m.Bar), { ssr: false });
const Cell = dynamic(() => import("recharts").then((m) => m.Cell), { ssr: false });
const XAxis = dynamic(() => import("recharts").then((m) => m.XAxis), { ssr: false });
const YAxis = dynamic(() => import("recharts").then((m) => m.YAxis), { ssr: false });
const CartesianGrid = dynamic(() => import("recharts").then((m) => m.CartesianGrid), { ssr: false });
const Tooltip = dynamic(() => import("recharts").then((m) => m.Tooltip), { ssr: false });
const Legend = dynamic(() => import("recharts").then((m) => m.Legend), { ssr: false });
const Label = dynamic(() => import("recharts").then((m) => m.Label), { ssr: false });

const BAR_COLORS = [
  "#1a1a1a", "#4a6fa5", "#b5838d", "#6b9e78", "#c9a96e",
  "#7b6fa5", "#d4846a", "#5b9caa", "#a07850", "#6b8c42",
  "#b56b6b", "#5b7b9c",
];

interface StatsProps {
  overview: ServerStatsOverviewResponse;
  overTime: ServerStatsPeriodsResponse;
  films: ServerStatsFilmsResponse;
  cameras: ServerStatsCamerasResponse;
  colors: ServerStatsColorsResponse;
}

function fmt(n?: number): string {
  if (n == null) return "—";
  return n.toLocaleString();
}

function fmtScore(n?: number): string {
  if (n == null) return "—";
  return n.toLocaleString(undefined, { maximumFractionDigits: 0 });
}

export default function StatsPage({ overview, overTime, films, cameras, colors }: StatsProps) {
  const d = overview.data;

  const overTimeData = (overTime.data ?? []).map((p) => ({
    date: p.period?.slice(0, 7) ?? "",
    count: p.count ?? 0,
    "average score": Math.round(p.avgScore ?? 0),
  }));

  const filmsData = (films.data ?? [])
    .slice(0, 12)
    .map((f) => ({
      name: `${f.filmMake ?? ""} ${f.filmType ?? ""}`.trim(),
      posts: f.postCount ?? 0,
    }));

  const camerasData = (cameras.data ?? [])
    .slice(0, 12)
    .map((c) => ({
      name: `${c.cameraMake ?? ""} ${c.cameraModel ?? ""}`.trim(),
      posts: c.postCount ?? 0,
    }));

  const colorsData = (colors.data ?? []).map((c) => ({
    name: c.htmlName ?? "",
    hex: c.hex ?? "#888",
    posts: c.postCount ?? 0,
  }));

  return (
    <div className={styles.page}>
      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>Overview</Title>
        <SimpleGrid cols={{ base: 2, sm: 3 }} spacing="md">
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>total posts</Text>
            <Text className={styles.tileValue}>{fmt(d?.totalPosts)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>authors</Text>
            <Text className={styles.tileValue}>{fmt(d?.totalAuthors)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>cameras</Text>
            <Text className={styles.tileValue}>{fmt(d?.totalCameras)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>films</Text>
            <Text className={styles.tileValue}>{fmt(d?.totalFilms)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>keywords</Text>
            <Text className={styles.tileValue}>{fmt(d?.totalKeywords)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>avg score</Text>
            <Text className={styles.tileValue}>{fmtScore(d?.avgScore)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>median score</Text>
            <Text className={styles.tileValue}>{fmtScore(d?.medianScore)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>min score</Text>
            <Text className={styles.tileValue}>{fmtScore(d?.minScore)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>max score</Text>
            <Text className={styles.tileValue}>{fmtScore(d?.maxScore)}</Text>
          </div>
          <div className={styles.tile}>
            <Text className={styles.tileLabel}>score std dev</Text>
            <Text className={styles.tileValue}>{fmtScore(d?.stdDevScore)}</Text>
          </div>
        </SimpleGrid>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>Posts Over Time</Title>
        <div className={styles.chartWrap}>
          <ResponsiveContainer width="100%" height={320}>
            <LineChart data={overTimeData} margin={{ top: 10, right: 60, left: 10, bottom: 30 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} interval="preserveStartEnd">
                <Label value="Date" offset={-15} position="insideBottom" style={{ fontSize: 12, fill: "#666" }} />
              </XAxis>
              <YAxis yAxisId="left" tick={{ fontSize: 11 }}>
                <Label value="posts" angle={-90} position="insideLeft" offset={15} style={{ fontSize: 12, fill: "#666" }} />
              </YAxis>
              <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 11 }}>
                <Label value="average score" angle={90} position="insideRight" offset={15} style={{ fontSize: 12, fill: "#888" }} />
              </YAxis>
              <Tooltip />
              <Legend layout="vertical" align="right" verticalAlign="middle" wrapperStyle={{ paddingLeft: 16, fontSize: 12 }} />
              <Line yAxisId="left" type="monotone" dataKey="count" stroke="#1a1a1a" dot={false} strokeWidth={2} />
              <Line yAxisId="right" type="monotone" dataKey="average score" stroke="#888" dot={false} strokeWidth={1.5} strokeDasharray="5 5" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>Top Films</Title>
        <div className={styles.chartWrap}>
          <ResponsiveContainer width="100%" height={Math.max(280, filmsData.length * 28)}>
            <BarChart layout="vertical" data={filmsData} margin={{ top: 5, right: 30, left: 10, bottom: 30 }}>
              <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="#e0e0e0" />
              <XAxis type="number" tick={{ fontSize: 11 }}>
                <Label value="posts" offset={-15} position="insideBottom" style={{ fontSize: 12, fill: "#666" }} />
              </XAxis>
              <YAxis type="category" dataKey="name" width={140} tick={{ fontSize: 11 }} />
              <Tooltip />
              <Bar dataKey="posts" radius={[0, 2, 2, 0]}>
                {filmsData.map((_, i) => (
                  <Cell key={i} fill={BAR_COLORS[i % BAR_COLORS.length]} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>Top Cameras</Title>
        <div className={styles.chartWrap}>
          <ResponsiveContainer width="100%" height={Math.max(280, camerasData.length * 28)}>
            <BarChart layout="vertical" data={camerasData} margin={{ top: 5, right: 30, left: 10, bottom: 30 }}>
              <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="#e0e0e0" />
              <XAxis type="number" tick={{ fontSize: 11 }}>
                <Label value="posts" offset={-15} position="insideBottom" style={{ fontSize: 12, fill: "#666" }} />
              </XAxis>
              <YAxis type="category" dataKey="name" width={140} tick={{ fontSize: 11 }} />
              <Tooltip />
              <Bar dataKey="posts" radius={[0, 2, 2, 0]}>
                {camerasData.map((_, i) => (
                  <Cell key={i} fill={BAR_COLORS[i % BAR_COLORS.length]} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>Top Colors</Title>
        <div className={styles.chartWrap}>
          <ResponsiveContainer width="100%" height={Math.max(280, colorsData.length * 22)}>
            <BarChart layout="vertical" data={colorsData} margin={{ top: 5, right: 30, left: 10, bottom: 30 }}>
              <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="#e0e0e0" />
              <XAxis type="number" tick={{ fontSize: 11 }}>
                <Label value="posts" offset={-15} position="insideBottom" style={{ fontSize: 12, fill: "#666" }} />
              </XAxis>
              <YAxis type="category" dataKey="name" width={100} tick={{ fontSize: 11 }} />
              <Tooltip />
              <Bar dataKey="posts" radius={[0, 2, 2, 0]}>
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
