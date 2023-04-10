import { expect } from "chai";

import { WALLET_PRIVATE_KEY, WORMHOLE_STATE_ID } from "./helpers/consts";
import {
  Ed25519Keypair,
  JsonRpcProvider,
  localnetConnection,
  RawSigner,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import { getPackageId } from "./helpers/utils";
import { addPrepareMessageAndPublishMessage } from "./helpers/wormhole/testPublishMessage";

describe(" 1. Wormhole", () => {
  const provider = new JsonRpcProvider(localnetConnection);

  // User wallet.
  const wallet = new RawSigner(
    Ed25519Keypair.fromSecretKey(WALLET_PRIVATE_KEY),
    provider
  );

  describe("Publish Message", () => {
    it("Check `WormholeMessage` Event", async () => {
      const wormholePackage = await getPackageId(
        wallet.provider,
        WORMHOLE_STATE_ID
      );

      const owner = await wallet.getAddress();

      // Create emitter cap.
      const emitterCapId = await (async () => {
        const tx = new TransactionBlock();
        const [emitterCap] = tx.moveCall({
          target: `${wormholePackage}::emitter::new`,
          arguments: [tx.object(WORMHOLE_STATE_ID)],
        });
        tx.transferObjects([emitterCap], tx.pure(owner));

        // Execute and fetch created Emitter cap.
        return wallet
          .signAndExecuteTransactionBlock({
            transactionBlock: tx,
            options: {
              showObjectChanges: true,
            },
          })
          .then((result) => {
            const found = result.objectChanges?.filter(
              (item) => "created" === item.type!
            );
            if (found?.length == 1 && "objectId" in found[0]) {
              return found[0].objectId;
            }

            throw new Error("no objects found");
          });
      })();

      // Publish messages using emitter cap.
      {
        const nonce = 69;
        const basePayload = "All your base are belong to us.";

        const numMessages = 32;
        const payloads: string[] = [];
        const tx = new TransactionBlock();

        // Construct transaction block to send multiple messages.
        for (let i = 0; i < numMessages; ++i) {
          // Make a unique message.
          const payload = basePayload + `... ${i}`;
          payloads.push(payload);

          addPrepareMessageAndPublishMessage(
            tx,
            wormholePackage,
            WORMHOLE_STATE_ID,
            emitterCapId,
            nonce,
            payload
          );
        }

        const events = await wallet
          .signAndExecuteTransactionBlock({
            transactionBlock: tx,
            options: {
              showEvents: true,
            },
          })
          .then((result) => result.events!);
        expect(events).has.length(numMessages);

        for (let i = 0; i < numMessages; ++i) {
          const eventData = events[i].parsedJson!;
          expect(eventData.consistency_level).equals(0);
          expect(eventData.nonce).equals(nonce);
          expect(eventData.payload).deep.equals([...Buffer.from(payloads[i])]);
          expect(eventData.sender).equals(emitterCapId);
          expect(eventData.sequence).equals(i.toString());
          expect(BigInt(eventData.timestamp) > 0n).is.true;
        }
      }
    });
  });
});
