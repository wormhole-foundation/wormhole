import { expect } from "chai";
import * as mock from "@certusone/wormhole-sdk/lib/cjs/mock";

import {
  CREATOR_PRIVATE_KEY,
  GUARDIAN_PRIVATE_KEY,
  RELAYER_PRIVATE_KEY,
  WALLET_PRIVATE_KEY,
} from "./helpers/consts";
import {
  Ed25519Keypair,
  JsonRpcProvider,
  localnetConnection,
  RawSigner,
} from "@mysten/sui.js";

describe(" 0. Environment", () => {
  const provider = new JsonRpcProvider(localnetConnection);

  // User wallet.
  const wallet = new RawSigner(
    Ed25519Keypair.fromSecretKey(WALLET_PRIVATE_KEY),
    provider
  );

  // Relayer wallet.
  const relayer = new RawSigner(
    Ed25519Keypair.fromSecretKey(RELAYER_PRIVATE_KEY),
    provider
  );

  // Deployer wallet.
  const creator = new RawSigner(
    Ed25519Keypair.fromSecretKey(CREATOR_PRIVATE_KEY),
    provider
  );

  describe("Verify Local Validator", () => {
    it("Balance", async () => {
      // Balance check wallet.
      {
        const coinData = await wallet
          .getAddress()
          .then((owner) =>
            provider
              .getCoins({ owner, coinType: "0x2::sui::SUI" })
              .then((result) => result.data)
          );
        expect(coinData).has.length(5);
      }

      // Balance check relayer.
      {
        const coinData = await relayer
          .getAddress()
          .then((owner) =>
            provider
              .getCoins({ owner, coinType: "0x2::sui::SUI" })
              .then((result) => result.data)
          );
        expect(coinData).has.length(5);
      }

      // Balance check creator. This should only have one gas object at this
      // point.
      {
        const coinData = await creator
          .getAddress()
          .then((owner) =>
            provider
              .getCoins({ owner, coinType: "0x2::sui::SUI" })
              .then((result) => result.data)
          );
        expect(coinData).has.length(1);
      }
    });
  });
});
