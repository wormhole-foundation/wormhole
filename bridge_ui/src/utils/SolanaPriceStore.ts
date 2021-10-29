import { MARKETS } from "@project-serum/serum";
import { Connection, PublicKey } from "@solana/web3.js";

export interface Markets {
  [coin: string]: {
    publicKey?: PublicKey;
    name: string;
    deprecated?: boolean;
  };
}

export const serumMarkets = (() => {
  const m: Markets = {};
  MARKETS.forEach((market) => {
    const coin = market.name.split("/")[0];
    if (m[coin]) {
      // Only override a market if it's not deprecated	.
      if (!m.deprecated) {
        m[coin] = {
          publicKey: market.address,
          name: market.name.split("/").join(""),
        };
      }
    } else {
      m[coin] = {
        publicKey: market.address,
        name: market.name.split("/").join(""),
      };
    }
  });

  m["USDC"] = m["USDT"];

  return m;
})();

// Create a cached API wrapper to avoid rate limits.
class PriceStore {
  cache: Map<String, number | undefined>;

  constructor() {
    this.cache = new Map();
  }

  async getPrice(
    connection: Connection,
    marketName: string
  ): Promise<number | undefined> {
    return new Promise((resolve, reject) => {
      if (this.cache.get(marketName) === undefined) {
        fetch(`https://serum-api.bonfida.com/orderbooks/${marketName}`).then(
          (resp) => {
            resp.json().then((resp) => {
              if (resp.data.asks === null || resp.data.bids === null) {
                resolve(undefined);
              } else if (
                resp.data.asks.length === 0 &&
                resp.data.bids.length === 0
              ) {
                resolve(undefined);
              } else if (resp.data.asks.length === 0) {
                resolve(resp.data.bids[0].price);
              } else if (resp.data.bids.length === 0) {
                resolve(resp.data.asks[0].price);
              } else {
                const mid =
                  (resp.data.asks[0].price + resp.data.bids[0].price) / 2.0;
                this.cache.set(marketName, mid);
                resolve(this.cache.get(marketName));
              }
            });
          }
        );
      } else {
        return resolve(this.cache.get(marketName));
      }
    });
  }
}

export const priceStore = new PriceStore();
