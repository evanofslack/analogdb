import { ReviewPage } from "@lib/adminClient";
import Pager from "./pager";
import ReviewGrid from "./reviewGrid";

interface ReviewTabProps {
  page: ReviewPage;
  firstPage: boolean;
}

export default function ReviewTab({ page, firstPage }: ReviewTabProps) {
  return (
    <>
      <ReviewGrid posts={page.posts} firstPage={firstPage} />
      <Pager
        newest={firstPage ? null : "/admin?tab=review"}
        older={
          page.next_cursor
            ? `/admin?tab=review&cursor=${encodeURIComponent(page.next_cursor)}`
            : null
        }
      />
    </>
  );
}
