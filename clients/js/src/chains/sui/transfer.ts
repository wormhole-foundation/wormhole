import { Transaction } from "@mysten/sui/transactions";
import {
  executeTransactionBlock,
  getPackageId,
  getProvider,
  getSigner,
  setMaxGasBudgetDevnet,
  SUI_CLOCK_OBJECT_ID,
} from "./utils";
import {
  Chain,
  Network,
  chainToChainId,
  contracts,
} from "@wormhole-foundation/sdk-base";
import { tryNativeToUint8Array } from "../../sdk/array";

const SUI_TYPE_ARG = "0x2::sui::SUI";

export async function transferSui(
  dstChain: Chain,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const core = contracts.coreBridge(network, "Sui");
  if (!core) {
    throw Error("Core bridge object ID is undefined");
  }
  const token_bridge = contracts.tokenBridge.get(network, "Sui");
  if (!token_bridge) {
    throw new Error("Token bridge object ID is undefined");
  }

  const client = getProvider(network, rpc);
  const signer = getSigner(client, network);
  const owner = signer.keypair.getPublicKey().toSuiAddress();
  const coinType = tokenAddress === "native" ? SUI_TYPE_ARG : tokenAddress;
  const recipientChainId = chainToChainId(dstChain);
  const recipient = tryNativeToUint8Array(dstAddress, recipientChainId);
  const amt = BigInt(amount);

  // Collect the sender's coins of the transfer type so they can be merged.
  const coinObjectIds: string[] = [];
  let cursor: string | null = null;
  let hasNextPage = true;
  while (hasNextPage) {
    const page = await client.listCoins({ owner, coinType, cursor });
    for (const coin of page.objects) {
      coinObjectIds.push(coin.objectId);
    }
    hasNextPage = page.hasNextPage;
    cursor = page.cursor;
  }
  if (coinObjectIds.length === 0) {
    throw new Error(`No coins of type ${coinType} found for ${owner}`);
  }

  const [coreBridgePackageId, tokenBridgePackageId] = await Promise.all([
    getPackageId(client, core),
    getPackageId(client, token_bridge),
  ]);

  const tx = new Transaction();

  const [transferCoin] = (() => {
    if (coinType === SUI_TYPE_ARG) {
      return tx.splitCoins(tx.gas, [tx.pure("u64", amt)]);
    }
    const [primaryCoinId, ...mergeCoinIds] = coinObjectIds;
    const primaryCoinInput = tx.object(primaryCoinId);
    if (mergeCoinIds.length) {
      tx.mergeCoins(
        primaryCoinInput,
        mergeCoinIds.map((id) => tx.object(id))
      );
    }
    return tx.splitCoins(primaryCoinInput, [tx.pure("u64", amt)]);
  })();

  const [feeCoin] = tx.splitCoins(tx.gas, [tx.pure("u64", BigInt(0))]);

  const [assetInfo] = tx.moveCall({
    target: `${tokenBridgePackageId}::state::verified_asset`,
    arguments: [tx.object(token_bridge)],
    typeArguments: [coinType],
  });

  // Random 32-bit transfer nonce.
  const nonce = Math.floor(Math.random() * 0xffffffff);

  const [transferTicket, dust] = tx.moveCall({
    target: `${tokenBridgePackageId}::transfer_tokens::prepare_transfer`,
    arguments: [
      assetInfo,
      transferCoin,
      tx.pure("u16", recipientChainId),
      tx.pure("vector<u8>", [...recipient]),
      tx.pure("u64", BigInt(0)),
      tx.pure("u32", nonce),
    ],
    typeArguments: [coinType],
  });

  tx.moveCall({
    target: `${tokenBridgePackageId}::coin_utils::return_nonzero`,
    arguments: [dust],
    typeArguments: [coinType],
  });

  const [messageTicket] = tx.moveCall({
    target: `${tokenBridgePackageId}::transfer_tokens::transfer_tokens`,
    arguments: [tx.object(token_bridge), transferTicket],
    typeArguments: [coinType],
  });

  tx.moveCall({
    target: `${coreBridgePackageId}::publish_message::publish_message`,
    arguments: [
      tx.object(core),
      feeCoin,
      messageTicket,
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });

  setMaxGasBudgetDevnet(network, tx);
  const result = await executeTransactionBlock(signer, tx);
  console.log(JSON.stringify(result));
}
