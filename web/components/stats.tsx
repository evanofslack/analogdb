"use client";

import { BarsList, DonutChart, LineChart } from "@mantine/charts";
import { SimpleGrid, Text, Title } from "@mantine/core";
import type {
  ServerStatsCamerasResponse,
  ServerStatsColorsResponse,
  ServerStatsFilmsResponse,
  ServerStatsOverviewResponse,
  ServerStatsPeriodsResponse,
} from "analogdb-generated";
import styles from "./stats.module.css";

const PIE_COLORS = [
  "#1a1a1a",
  "#4a6fa5",
  "#b5838d",
  "#6b9e78",
  "#c9a96e",
  "#7b6fa5",
  "#d4846a",
  "#5b9caa",
  "#a07850",
  "#6b8c42",
  "#b56b6b",
  "#5b7b9c",
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

interface TileProps {
  label: string;
  value: string;
}

function Tile({ label, value }: TileProps) {
  return (
    <div className={styles.tile}>
      <Text className={styles.tileLabel}>{label}</Text>
      <Text className={styles.tileValue}>{value}</Text>
    </div>
  );
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
    date: p.period?.slice(0, 7) ?? "",
    count: p.count ?? 0,
    "average score": Math.round(p.avgScore ?? 0),
  }));

  const filmsData = (films.data ?? []).slice(0, 12).map((f, i) => ({
    name: `${f.filmMake ?? ""} ${f.filmType ?? ""}`.trim(),
    value: f.postCount ?? 0,
    color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  const camerasData = (cameras.data ?? []).slice(0, 12).map((c, i) => ({
    name: `${c.cameraMake ?? ""} ${c.cameraModel ?? ""}`.trim(),
    value: c.postCount ?? 0,
    color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  const colorsData = (colors.data ?? []).map((c) => ({
    name: c.htmlName ?? "",
    value: c.postCount ?? 0,
    color: c.hex ?? "#888",
  }));

  return (
    <div className={styles.page}>
      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Overview
        </Title>

        <SimpleGrid cols={3} spacing="md" className={styles.tileGroup}>
          <Tile label="total posts" value={fmt(d?.totalPosts)} />
          <Tile label="authors" value={fmt(d?.totalAuthors)} />
          <Tile label="keywords" value={fmt(d?.totalKeywords)} />
        </SimpleGrid>

        <SimpleGrid cols={2} spacing="md" className={styles.tileGroup}>
          <Tile label="films" value={fmt(d?.totalFilms)} />
          <Tile label="cameras" value={fmt(d?.totalCameras)} />
        </SimpleGrid>

        <SimpleGrid cols={5} spacing="md" className={styles.tileGroup}>
          <Tile label="min score" value={fmtScore(d?.minScore)} />
          <Tile label="median score" value={fmtScore(d?.medianScore)} />
          <Tile label="avg score" value={fmtScore(d?.avgScore)} />
          <Tile label="max score" value={fmtScore(d?.maxScore)} />
          <Tile label="std dev" value={fmtScore(d?.stdDevScore)} />
        </SimpleGrid>
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Posts Over Time
        </Title>
        <LineChart
          h={320}
          data={overTimeData}
          dataKey="date"
          withDots={false}
          withRightYAxis
          xAxisLabel="date"
          yAxisLabel="posts"
          rightYAxisLabel="average score"
          withLegend
          legendProps={{
            verticalAlign: "top",
            align: "right",
            layout: "vertical",
            wrapperStyle: { paddingLeft: 16, fontSize: 12 },
          }}
          series={[
            {
              name: "count",
              color: "#1a1a1a",
              yAxisId: "left",
              label: "posts",
            },
            {
              name: "average score",
              color: "#888",
              yAxisId: "right",
              strokeDasharray: "5 5",
            },
          ]}
        />
      </section>

      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Top Films, Cameras &amp; Colors
        </Title>
        <div className={styles.pieRow}>
          <BarsList
            data={filmsData}
            minBarSize={200}
            valueFormatter={(value) => value.toLocaleString("en-US")}
            barsLabel="Films"
            valueLabel="Posts"
          />
          <BarsList
            data={camerasData}
            minBarSize={100}
            valueFormatter={(value) => value.toLocaleString("en-US")}
            barsLabel="Cameras"
            valueLabel="Posts"
          />
          <BarsList
            data={colorsData}
            minBarSize={50}
            valueFormatter={(value) => value.toLocaleString("en-US")}
            barsLabel="Colors"
            valueLabel="Posts"
          />
        </div>
      </section>
    </div>
  );
}
