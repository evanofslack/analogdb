import { formatDateTime } from "@lib/format";
import { Button } from "@mantine/core";
import Link from "next/link";
import styles from "./adminPanel.module.css";

export default function AuditTab({ page }) {
  return (
    <>
      <p className={styles.note} style={{ marginBottom: "1rem" }}>
        Authorized writes and failed logins (401) on the API. The web server and
        the scraper share credentials, the client column tells them apart.
      </p>
      {page.entries.length === 0 ? (
        <p className={styles.note}>No entries.</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Time</th>
                <th>Method</th>
                <th>Path</th>
                <th className={styles.num}>Status</th>
                <th>Client</th>
                <th>IP</th>
                <th>User agent</th>
                <th>Request id</th>
              </tr>
            </thead>
            <tbody>
              {page.entries.map((e) => (
                <tr
                  key={e.request_id || e.start_ms}
                  className={e.status === 401 ? styles.errorRow : undefined}
                >
                  <td>{formatDateTime(e.time)}</td>
                  <td>{e.method}</td>
                  <td className={styles.mono}>{e.path}</td>
                  <td className={styles.num}>{e.status}</td>
                  <td>{e.client}</td>
                  <td className={styles.mono}>{e.remote_ip}</td>
                  <td>{e.user_agent}</td>
                  <td className={styles.mono}>{e.request_id}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className={styles.pager}>
        {page.next_before && (
          <Button
            component={Link}
            href={`/admin?tab=audit&before=${page.next_before}`}
            variant="default"
          >
            Older
          </Button>
        )}
      </div>
    </>
  );
}
