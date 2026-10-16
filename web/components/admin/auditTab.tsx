import { AuditPage } from "@lib/adminClient";
import { formatDateTime } from "@lib/format";
import styles from "./adminPanel.module.css";
import Pager from "./pager";

interface AuditTabProps {
  page: AuditPage;
  firstPage: boolean;
}

export default function AuditTab({ page, firstPage }: AuditTabProps) {
  return (
    <>
      {page.entries.length === 0 ? (
        <p className={styles.empty}>No entries</p>
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
                  className={
                    e.status === 401 || e.status === 403
                      ? styles.errorRow
                      : undefined
                  }
                >
                  <td className={styles.nowrap}>{formatDateTime(e.time)}</td>
                  <td>{e.method}</td>
                  <td className={styles.mono}>{e.path}</td>
                  <td className={styles.num}>{e.status}</td>
                  <td>{e.client}</td>
                  <td className={styles.mono}>{e.remote_ip}</td>
                  <td className={styles.wrap}>{e.user_agent}</td>
                  <td className={styles.mono}>{e.request_id}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <Pager
        newest={firstPage ? null : "/admin?tab=audit"}
        older={
          page.next_before
            ? `/admin?tab=audit&before=${page.next_before}`
            : null
        }
      />
    </>
  );
}
