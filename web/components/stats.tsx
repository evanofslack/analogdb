"use client";

import { BarsList, LineChart } from "@mantine/charts";
import { Text, Title } from "@mantine/core";
import type {
  ServerStatsCamerasResponse,
  ServerStatsColorsResponse,
  ServerStatsFilmsResponse,
  ServerStatsKeywordsResponse,
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
  filmsByCount: ServerStatsFilmsResponse;
  filmsByScore: ServerStatsFilmsResponse;
  camerasByCount: ServerStatsCamerasResponse;
  camerasByScore: ServerStatsCamerasResponse;
  colorsByCount: ServerStatsColorsResponse;
  colorsByScore: ServerStatsColorsResponse;
  keywordsByCount: ServerStatsKeywordsResponse;
  keywordsByScore: ServerStatsKeywordsResponse;
}

function fmt(n?: number): string {
  if (n == null) return "—";
  return n.toLocaleString();
}

function fmtScore(n?: number): string {
  if (n == null) return "—";
  return n.toLocaleString(undefined, { maximumFractionDigits: 0 });
}

interface StatGroupProps {
  items: { label: string; value: string }[];
}

function StatGroup({ items }: StatGroupProps) {
  return (
    <div className={styles.statGroup}>
      {items.map((item, i) => (
        <div key={item.label} className={styles.statGroupItem}>
          {i > 0 && <div className={styles.statDivider} />}
          <div className={styles.statContent}>
            <Text className={styles.tileLabel}>{item.label}</Text>
            <Text className={styles.tileValue}>{item.value}</Text>
          </div>
        </div>
      ))}
    </div>
  );
}

interface ChartPairProps {
  title: string;
  leftLabel: string;
  rightLabel: string;
  leftData: { name: string; value: number; color: string }[];
  rightData: { name: string; value: number; color: string }[];
  leftValueFormatter?: (v: number) => string;
  rightValueFormatter?: (v: number) => string;
}

function ChartPair({
  title,
  leftLabel,
  rightLabel,
  leftData,
  rightData,
  leftValueFormatter,
  rightValueFormatter,
}: ChartPairProps) {
  return (
    <section className={styles.section}>
      <Title order={2} className={styles.sectionTitle}>
        {title}
      </Title>
      <div className={styles.chartPair}>
        <div className={styles.chartHalf}>
          <Text className={styles.chartHalfTitle}>{leftLabel}</Text>
          <BarsList
            data={leftData}
            valueFormatter={leftValueFormatter ?? ((v) => v.toLocaleString())}
          />
        </div>
        <div className={styles.chartHalf}>
          <Text className={styles.chartHalfTitle}>{rightLabel}</Text>
          <BarsList
            data={rightData}
            valueFormatter={rightValueFormatter ?? ((v) => fmtScore(v))}
          />
        </div>
      </div>
    </section>
  );
}

export default function StatsPage({
  overview,
  overTime,
  filmsByCount,
  filmsByScore,
  camerasByCount,
  camerasByScore,
  colorsByCount,
  colorsByScore,
  keywordsByCount,
  keywordsByScore,
}: StatsProps) {
  const d = overview.data;

  const overTimeData = (overTime.data ?? []).map((p) => ({
    date: p.period?.slice(0, 7) ?? "",
    count: p.count ?? 0,
    "average score": Math.round(p.avgScore ?? 0),
  }));

  const filmsCountData = (filmsByCount.data ?? []).map((f, i) => ({
    name: `${f.filmMake ?? ""} ${f.filmType ?? ""}`.trim(),
    value: f.postCount ?? 0,
    color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  const filmsScoreData = (filmsByScore.data ?? []).map((f, i) => ({
    name: `${f.filmMake ?? ""} ${f.filmType ?? ""}`.trim(),
    value: Math.round(f.avgScore ?? 0),
    color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  const camerasCountData = (camerasByCount.data ?? []).map((c, i) => ({
    name: `${c.cameraMake ?? ""} ${c.cameraModel ?? ""}`.trim(),
    value: c.postCount ?? 0,
    color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  const camerasScoreData = (camerasByScore.data ?? []).map((c, i) => ({
    name: `${c.cameraMake ?? ""} ${c.cameraModel ?? ""}`.trim(),
    value: Math.round(c.avgScore ?? 0),
    color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  const colorsCountData = (colorsByCount.data ?? []).map((c) => ({
    name: c.htmlName ?? "",
    value: c.postCount ?? 0,
    color: c.hex ?? "#888",
  }));

  const colorsScoreData = (colorsByScore.data ?? []).map((c) => ({
    name: c.htmlName ?? "",
    value: Math.round(c.avgScore ?? 0),
    color: c.hex ?? "#888",
  }));

  const keywordsCountData = (keywordsByCount.data ?? []).map((k, i) => ({
    name: k.word ?? "",
    value: k.postCount ?? 0,
    color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  const keywordsScoreData = (keywordsByScore.data ?? []).map((k, i) => ({
    name: k.word ?? "",
    value: Math.round(k.avgScore ?? 0),
    color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  return (
    <div className={styles.page}>
      <section className={styles.section}>
        <Title order={2} className={styles.sectionTitle}>
          Overview
        </Title>

        <StatGroup
          items={[
            { label: "posts", value: fmt(d?.totalPosts) },
            { label: "authors", value: fmt(d?.totalAuthors) },
            { label: "keywords", value: fmt(d?.totalKeywords) },
          ]}
        />

        <StatGroup
          items={[
            { label: "film brands", value: fmt(d?.totalFilmBrands) },
            { label: "film stocks", value: fmt(d?.totalFilmStocks) },
            { label: "camera brands", value: fmt(d?.totalCameraBrands) },
            { label: "camera models", value: fmt(d?.totalCameraModels) },
          ]}
        />

        <StatGroup
          items={[
            { label: "min score", value: fmtScore(d?.minScore) },
            { label: "median score", value: fmtScore(d?.medianScore) },
            { label: "avg score", value: fmtScore(d?.avgScore) },
            { label: "max score", value: fmtScore(d?.maxScore) },
            { label: "std dev", value: fmtScore(d?.stdDevScore) },
          ]}
        />
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

      <ChartPair
        title="Films"
        leftLabel="Most shot"
        rightLabel="Highest rated"
        leftData={filmsCountData}
        rightData={filmsScoreData}
        leftValueFormatter={(v) => v.toLocaleString() + " posts"}
        rightValueFormatter={(v) => "avg " + fmtScore(v)}
      />

      <ChartPair
        title="Cameras"
        leftLabel="Most shot"
        rightLabel="Highest rated"
        leftData={camerasCountData}
        rightData={camerasScoreData}
        leftValueFormatter={(v) => v.toLocaleString() + " posts"}
        rightValueFormatter={(v) => "avg " + fmtScore(v)}
      />

      <ChartPair
        title="Colors"
        leftLabel="Most common"
        rightLabel="Highest rated"
        leftData={colorsCountData}
        rightData={colorsScoreData}
        leftValueFormatter={(v) => v.toLocaleString() + " posts"}
        rightValueFormatter={(v) => "avg " + fmtScore(v)}
      />

      <ChartPair
        title="Keywords"
        leftLabel="Most used"
        rightLabel="Highest rated"
        leftData={keywordsCountData}
        rightData={keywordsScoreData}
        leftValueFormatter={(v) => v.toLocaleString() + " posts"}
        rightValueFormatter={(v) => "avg " + fmtScore(v)}
      />
    </div>
  );
}
