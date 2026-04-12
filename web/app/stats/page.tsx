import {
  getStatsCameras,
  getStatsColors,
  getStatsFilms,
  getStatsKeywords,
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

  const [
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
  ] = await Promise.all([
    getStatsOverview({}),
    getStatsPostsOverTime({ granularity: "month", start: 1640995200 }),
    getStatsFilms({ limit: 12, metric: "count" }),
    getStatsFilms({ limit: 12, metric: "score" }),
    getStatsCameras({ limit: 12, metric: "count" }),
    getStatsCameras({ limit: 12, metric: "score" }),
    getStatsColors({ limit: 12, metric: "count" }),
    getStatsColors({ limit: 12, metric: "score" }),
    getStatsKeywords({ limit: 12, metric: "count" }),
    getStatsKeywords({ limit: 12, metric: "score" }),
  ]);

  return (
    <div className={styles.container}>
      <Header isAdmin={isAdmin} />
      <StatsPage
        overview={overview}
        overTime={overTime}
        filmsByCount={filmsByCount}
        filmsByScore={filmsByScore}
        camerasByCount={camerasByCount}
        camerasByScore={camerasByScore}
        colorsByCount={colorsByCount}
        colorsByScore={colorsByScore}
        keywordsByCount={keywordsByCount}
        keywordsByScore={keywordsByScore}
      />
    </div>
  );
}
