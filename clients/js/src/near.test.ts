import { parseSeedPhrase } from "near-seed-phrase";
import { test } from "@jest/globals";
import { KeyPair } from "near-api-js";
import { keyPairToImplicitAccount } from "./near";
import base58 from "bs58";

test("keyPairToImplicitAccount", () => {
  // seed phrase from /near/devnet_deploy.ts
  const parsed = parseSeedPhrase(
    "weather opinion slam purpose access artefact word orbit matter rice poem badge"
  );
  const expectedImplicitAccount = base58
    .decode(parsed.publicKey.split(":")[1])
    .toString("hex");

  const keyPair = KeyPair.fromString(parsed.secretKey);
  const implicitAccount = keyPairToImplicitAccount(keyPair);
  expect(implicitAccount.length).toEqual(64);
  expect(implicitAccount).toEqual(expectedImplicitAccount);
});
