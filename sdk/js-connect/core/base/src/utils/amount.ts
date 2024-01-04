/**
 * Converts a human friendly decimal number to base units as an integer
 *
 * @param amount The decimal number as a string to convert into base units
 * @param decimals the number of decimals to normalize to
 * @returns The amount converted to base units as a BigNumber
 */
export function normalizeAmount(amount: number | string, decimals: bigint): bigint {
  // If we're passed a number, convert it to a string first
  // so we can do everything as bigints
  if (typeof amount === "number") amount = amount.toPrecision();

  // punting
  if (amount.includes("e")) throw new Error(`Exponential detected:  ${amount}`);

  // some slightly sketchy string manip

  const chunks = amount.split(".");
  if (chunks.length > 2) throw "Too many decimals";

  const [whole, partial] =
    chunks.length === 0 ? ["0", ""] : chunks.length === 1 ? [chunks[0], ""] : chunks;

  if (partial && partial.length > decimals)
    throw new Error(`Overspecified decimal amount: ${partial.length} > ${decimals}`);

  // combine whole and partial without decimals
  const amt = BigInt(whole + partial);

  // adjust number of decimals to account for decimals accounted for
  // when we remove the decimal place for amt
  decimals -= BigInt(partial.length);

  // finally, produce the number in base units
  return amt * 10n ** decimals;
}

/**
 * Converts a bigint amount to a friendly decimal number as a string
 *
 * @param amount The number of units as a bigint to convert into the display amount
 * @param decimals the number of decimals in the displayAmount
 * @returns The amount converted to a nice display string
 */
export function displayAmount(amount: bigint, decimals: bigint, displayDecimals: bigint): string {
  // first scale to remove any partial amounts but allowing for full
  // precision required by displayDecimals
  const amt = amount / 10n ** (decimals - displayDecimals);
  const numDec = Number(displayDecimals);
  // Final scaling then use the builtin `Number.tofixed` for formatting display amount
  return (Number(amt) / 10 ** numDec).toFixed(numDec);
}
