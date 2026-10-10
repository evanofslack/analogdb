// Report reasons a visitor can pick, matching analogdb.ReportReason in the backend
export const reportReasons = [
  { value: "takedown", label: "I own this photo and want it taken down" },
  { value: "nsfw_mislabeled", label: "This image is NSFW but marked as safe" },
  { value: "wrong_info", label: "The camera, film or other info is wrong" },
  { value: "not_film", label: "This is not a film photograph" },
  { value: "other", label: "Something else" },
] as const;

export type ReportReason = (typeof reportReasons)[number]["value"];

export const maxReportMessage = 500;
export const maxReportEmail = 254;

export function isReportReason(value: unknown): value is ReportReason {
  return reportReasons.some((r) => r.value === value);
}

export function reportReasonLabel(value: string): string {
  return reportReasons.find((r) => r.value === value)?.label ?? value;
}
