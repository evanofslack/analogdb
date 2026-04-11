import {
  getStatsCameras,
  getStatsColors,
  getStatsFilms,
  getStatsOverview,
  getStatsPostsOverTime,
} from "@app/actions/stats";
import StatsPage from "@components/stats";
import styles from "@components/gallery.module.css";
import Header from "@components/header";
import { checkAdminAuth } from "@lib/auth";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "AnalogDB — Stats",
  description: "Analytics and statistics for AnalogDB",
};

export const revalidate = 3600;

export default async function StatsPageRoute() {
  const isAdmin = await checkAdminAuth();

  const [overview, overTime, films, cameras, colors] = await Promise.all([
    getStatsOverview({}),
    getStatsPostsOverTime({ granularity: "month", start: 1640995200 }), // 2022-01-01
    getStatsFilms({ limit: 20, metric: "count" }),
    getStatsCameras({ limit: 20, metric: "count" }),
    getStatsColors({ limit: 30 }),
  ]);

  return (
    <div className={styles.container}>
      <Header isAdmin={isAdmin} />
      <StatsPage
        overview={overview}
        overTime={overTime}
        films={films}
        cameras={cameras}
        colors={colors}
      />
    </div>
  );
}
