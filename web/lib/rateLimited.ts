export type Limited = { rateLimited: true; retryAfter: number };

export class RateLimitedError extends Error {
  retryAfter: number;

  constructor(retryAfter: number) {
    super("too many requests, try again in a moment");
    this.name = "RateLimitedError";
    this.retryAfter = retryAfter;
  }
}

export function isLimited(value: unknown): value is Limited {
  return (
    typeof value === "object" &&
    value !== null &&
    (value as Limited).rateLimited === true
  );
}

export function unwrap<T>(result: T | Limited): T {
  if (isLimited(result)) throw new RateLimitedError(result.retryAfter);
  return result;
}

export function retryUnlessLimited(failureCount: number, error: Error) {
  return !(error instanceof RateLimitedError) && failureCount < 3;
}
