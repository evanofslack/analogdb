// Report reasons a visitor can pick, matching analogdb.ReportReason in the backend
export const reportReasons = [
  { value: "takedown", label: "I own this photo and want it taken down" },
  { value: "nsfw_mislabeled", label: "This image is NSFW but marked as safe" },
  { value: "wrong_info", label: "The camera, film or other info is wrong" },
  { value: "not_film", label: "This is not a film photograph" },
  { value: "duplicate", label: "This photo is already on AnalogDB" },
  { value: "other", label: "Something else" },
] as const;

type ReportReason = (typeof reportReasons)[number]["value"];

export const maxReportMessage = 500;
export const maxReportEmail = 254;

export function isReportReason(value: unknown): value is ReportReason {
  return reportReasons.some((r) => r.value === value);
}

export function reportReasonLabel(value: string): string {
  return reportReasons.find((r) => r.value === value)?.label ?? value;
}

// redditLink turns a stored permalink into a full link
export function redditLink(permalink: string | null): string | null {
  if (!permalink) return null;
  return permalink.startsWith("http")
    ? permalink
    : `https://www.reddit.com${permalink}`;
}
