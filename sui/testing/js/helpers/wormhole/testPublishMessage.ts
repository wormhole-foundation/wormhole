import { SUI_CLOCK_OBJECT_ID, TransactionBlock } from "@mysten/sui.js";

export function addPrepareMessageAndPublishMessage(
  tx: TransactionBlock,
  wormholePackage: string,
  wormholeStateId: string,
  emitterCapId: string,
  nonce: number,
  payload: number[] | string
): TransactionBlock {
  const [feeAmount] = tx.moveCall({
    target: `${wormholePackage}::state::message_fee`,
    arguments: [tx.object(wormholeStateId)],
  });
  const [wormholeFee] = tx.splitCoins(tx.gas, [feeAmount]);
  const [messageTicket] = tx.moveCall({
    target: `${wormholePackage}::publish_message::prepare_message`,
    arguments: [tx.object(emitterCapId), tx.pure(nonce), tx.pure(payload)],
  });
  tx.moveCall({
    target: `${wormholePackage}::publish_message::publish_message`,
    arguments: [
      tx.object(wormholeStateId),
      wormholeFee,
      messageTicket,
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });

  return tx;
}
