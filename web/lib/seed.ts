export const maxSeed = 500;

export function pickSeed(): number {
  return Math.floor(Math.random() * maxSeed) + 1;
}

export function isValidSeed(seed: number | null | undefined): boolean {
  return Number.isInteger(seed) && seed > 0 && seed <= 2147483647;
}
