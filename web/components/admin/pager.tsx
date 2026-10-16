import Link from "next/link";
import styles from "./adminPanel.module.css";

interface PagerProps {
  newest?: string | null;
  older?: string | null;
}

export default function Pager({ newest, older }: PagerProps) {
  if (!newest && !older) return null;
  return (
    <nav className={styles.pager}>
      {newest && <Link href={newest}>← Newest</Link>}
      {older && <Link href={older}>Older →</Link>}
    </nav>
  );
}
