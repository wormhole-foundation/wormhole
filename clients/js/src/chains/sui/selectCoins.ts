export interface CoinBalance {
  objectId: string;
  balance: bigint;
}

/**
 * Pick the fewest coins whose combined balance covers `amount`, largest first.
 * Keeps the PTB's `mergeCoins` list minimal so dust cannot bloat it. Throws if
 * the balances cannot cover `amount`.
 */
export const selectCoinsForAmount = (
  coins: readonly CoinBalance[],
  amount: bigint
): string[] => {
  const sorted = [...coins].sort((a, b) =>
    a.balance === b.balance ? 0 : a.balance > b.balance ? -1 : 1
  );

  // Always keeps at least one coin: the PTB needs a primary coin to split from.
  const selected: string[] = [];
  let total = BigInt(0);
  for (const coin of sorted) {
    selected.push(coin.objectId);
    total += coin.balance;
    if (total >= amount) {
      return selected;
    }
  }

  throw new Error(
    `Insufficient balance: have ${total}, need ${amount} (across ${coins.length} coin(s))`
  );
};
