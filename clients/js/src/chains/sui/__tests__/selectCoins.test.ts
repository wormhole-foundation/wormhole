import { selectCoinsForAmount } from "../selectCoins";

const coin = (objectId: string, balance: bigint) => ({ objectId, balance });

describe("selectCoinsForAmount", () => {
  it("uses a single coin when it already covers the amount", () => {
    const coins = [coin("a", BigInt(100)), coin("b", BigInt(50))];
    expect(selectCoinsForAmount(coins, BigInt(80))).toEqual(["a"]);
  });

  it("selects largest-first and stops once covered", () => {
    const coins = [
      coin("small", BigInt(1)),
      coin("big", BigInt(100)),
      coin("mid", BigInt(40)),
    ];
    expect(selectCoinsForAmount(coins, BigInt(120))).toEqual(["big", "mid"]);
  });

  it("ignores dust once the amount is covered", () => {
    // 500 dust coins plus one large coin: only the large one is needed.
    const dust = Array.from({ length: 500 }, (_, i) =>
      coin(`dust${i}`, BigInt(1))
    );
    const coins = [...dust, coin("whale", BigInt(1000))];

    const selected = selectCoinsForAmount(coins, BigInt(900));

    expect(selected).toEqual(["whale"]);
  });

  it("still merges several coins when no single one suffices", () => {
    const coins = [
      coin("a", BigInt(30)),
      coin("b", BigInt(30)),
      coin("c", BigInt(30)),
    ];
    expect(selectCoinsForAmount(coins, BigInt(61))).toEqual(["a", "b", "c"]);
  });

  it("keeps one coin for a zero-amount transfer", () => {
    const coins = [coin("a", BigInt(5)), coin("b", BigInt(10))];
    expect(selectCoinsForAmount(coins, BigInt(0))).toEqual(["b"]);
  });

  it("throws when the balances cannot cover the amount", () => {
    const coins = [coin("a", BigInt(10)), coin("b", BigInt(5))];
    expect(() => selectCoinsForAmount(coins, BigInt(100))).toThrow(
      /Insufficient balance: have 15, need 100/
    );
  });

  it("does not mutate the caller's array", () => {
    const coins = [coin("a", BigInt(1)), coin("b", BigInt(100))];
    const order = coins.map((c) => c.objectId);

    selectCoinsForAmount(coins, BigInt(50));

    expect(coins.map((c) => c.objectId)).toEqual(order);
  });
});
