"use client";

import { formatNumber } from "@lib/format";
import { ChartTooltip, LineChart } from "@mantine/charts";
import styles from "./adminPanel.module.css";

const day = 24 * 60 * 60 * 1000;

function bucketLabel(time, bucket, { full = false, multiDay = false } = {}) {
  const iso = new Date(time).toISOString();
  const date = full ? iso.slice(0, 10) : iso.slice(5, 10);
  if (bucket !== "hour") {
    return date;
  }
  const hour = `${iso.slice(11, 13)}:00`;
  if (full) {
    return `${date} ${hour} UTC`;
  }
  return multiDay ? `${date} ${hour}` : hour;
}

export default function TimeChart({ data, series, bucket, height = 220 }) {
  if (data.length === 0) {
    return <p className={styles.note}>No data in this range.</p>;
  }
  const span = new Date(data[data.length - 1].time) - new Date(data[0].time);
  const multiDay = span > day;
  return (
    <div className={styles.card}>
      <LineChart
        h={height}
        data={data}
        dataKey="time"
        series={series}
        curveType="linear"
        withDots={false}
        withLegend={series.length > 1}
        gridAxis="y"
        strokeWidth={1.5}
        valueFormatter={formatNumber}
        yAxisProps={{ width: 48 }}
        xAxisProps={{
          minTickGap: 24,
          tickFormatter: (time) => bucketLabel(time, bucket, { multiDay }),
        }}
        tooltipProps={{
          content: ({ label, payload }) => (
            <ChartTooltip
              label={label ? bucketLabel(label, bucket, { full: true }) : null}
              payload={payload}
              series={series}
              valueFormatter={formatNumber}
            />
          ),
        }}
      />
    </div>
  );
}
