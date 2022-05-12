export function safeBigIntToNumber(b: bigint): number {
  if (
    b < BigInt(Number.MIN_SAFE_INTEGER) ||
    b > BigInt(Number.MAX_SAFE_INTEGER)
  ) {
    throw new Error("integer is unsafe");
  }
  return Number(b);
}
