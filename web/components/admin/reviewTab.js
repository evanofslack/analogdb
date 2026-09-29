import { Button } from "@mantine/core";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import ReviewGrid from "./reviewGrid";

export default function ReviewTab({ page, firstPage }) {
  return (
    <>
      <ReviewGrid posts={page.posts} firstPage={firstPage} />
      <div className={styles.pager}>
        {!firstPage && (
          <Button
            component={Link}
            href="/admin?tab=review"
            variant="default"
            mr="sm"
          >
            Newest
          </Button>
        )}
        {page.next_cursor && (
          <Button
            component={Link}
            href={`/admin?tab=review&cursor=${encodeURIComponent(
              page.next_cursor
            )}`}
            variant="default"
          >
            Older
          </Button>
        )}
      </div>
    </>
  );
}
