/**
 * Converts a human friendly decimal number to base units as an integer
 *
 * @param amount The decimal number as a string to convert into base units
 * @param decimals the number of decimals to normalize to
 * @returns The amount converted to base units as a BigNumber
 */
export function normalizeAmount(
  amount: number | string,
  decimals: bigint,
): bigint {
  // If we're passed a number, convert it to a string first
  // so we can do everything as bigints
  if (typeof amount === "number") amount = amount.toPrecision();

  // punting
  if (amount.includes("e")) throw new Error(`Exponential detected:  ${amount}`);

  // If its a whole number, just add a decimal place to normalize
  if (!amount.includes(".")) amount += ".0";

  // some slightly sketchy
  const [whole, partial] = amount.split(".");
  if (partial.length > decimals)
    throw new Error(
      `Overspecified decimal amount: ${partial.length} > ${decimals}`,
    );

  // combine whole and partial without decimals
  const amt = BigInt(whole + partial);

  // adjust number of decimals to account for decimals accounted for
  // when we remove the decimal place for amt
  decimals -= BigInt(partial.length);

  // finally, produce the number in base units
  return amt * 10n ** decimals;
}
