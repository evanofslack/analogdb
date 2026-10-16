import Header from "@components/header";
import FamilySection from "@components/stats/familySection";
import LookSection from "@components/stats/lookSection";
import MethodSection from "@components/stats/methodSection";
import PopularitySection from "@components/stats/popularitySection";
import PostingSection from "@components/stats/postingSection";
import styles from "@components/stats/stats.module.css";
import StatsHeader from "@components/stats/statsHeader";
import SubjectSection from "@components/stats/subjectSection";
import TrendSection from "@components/stats/trendSection";
import { getStats } from "@lib/data/stats";
import { Metadata } from "next";
import { connection } from "next/server";

const title = "Stats";
const description =
  "How film stocks look, what's rising and what does well, measured from 24,000+ photos in AnalogDB.";

export const metadata: Metadata = {
  title,
  description,
  alternates: { canonical: "/stats" },
  openGraph: { siteName: "AnalogDB", title, description, url: "/stats" },
};

export default async function StatsPage() {
  await connection();
  const stats = await getStats();
  const films = stats?.films ?? [];

  return (
    <div className={styles.main}>
      <Header />
      <main className={styles.page}>
        <StatsHeader stats={stats} />
        {stats && (
          <>
            <LookSection films={films} />
            <FamilySection films={films} families={stats.families ?? []} />
            <SubjectSection stats={stats} />
            <TrendSection stats={stats} />
            <PopularitySection stats={stats} />
            <PostingSection stats={stats} />
            <MethodSection stats={stats} />
          </>
        )}
      </main>
    </div>
  );
}
