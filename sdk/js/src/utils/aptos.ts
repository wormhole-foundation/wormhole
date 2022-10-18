import { hexZeroPad } from "ethers/lib/utils";
import { sha3_256 } from "js-sha3";
import { ChainId, CHAIN_ID_APTOS, ensureHexPrefix, hex } from "../utils";
import { AptosAccount, AptosClient, Types } from "aptos";
import { State } from "../aptos/types";

export const signAndSubmitTransaction = (
  client: AptosClient,
  sender: AptosAccount,
  payload: Types.EntryFunctionPayload,
  opts?: Partial<Types.SubmitTransactionRequest>
): Promise<Types.PendingTransaction> => {
  // overwriting `max_gas_amount` default
  // rest of defaults are defined here: https://aptos-labs.github.io/ts-sdk-doc/classes/AptosClient.html#generateTransaction
  const customOpts = Object.assign(
    {
      gas_unit_price: "100",
      max_gas_amount: "30000",
    },
    opts
  );

  return (
    client
      // create raw transaction
      .generateTransaction(sender.address(), payload, customOpts)
      // simulate transaction
      .then((rawTx) =>
        client
          .simulateTransaction(sender, rawTx)
          .then((sims) =>
            sims.forEach((tx) => {
              if (!tx.success) {
                console.error(JSON.stringify(tx, null, 2));
                throw new Error(`Transaction failed: ${tx.vm_status}`);
              }
            })
          )
          .then((_) => rawTx)
      )
      // sign & submit transaction if simulation is successful
      .then((rawTx) => client.signTransaction(sender, rawTx))
      .then((signedTx) => client.submitTransaction(signedTx))
  );
};

/**
 * Create a transaction using the given payload and commit it on-chain.
 *
 * This functionality can be replicated by calling
 * `signAndSubmitTransaction(...).then(tx => client.waitForTransactionWithResult(tx.hash))`.
 * @param client
 * @param sender
 * @param payload
 * @returns Transaction info
 */
export const waitForSignAndSubmitTransaction = (
  client: AptosClient,
  sender: AptosAccount,
  payload: Types.EntryFunctionPayload
): Promise<Types.UserTransaction> => {
  return signAndSubmitTransaction(client, sender, payload).then(
    (pendingTx) =>
      client.waitForTransactionWithResult(
        pendingTx.hash
      ) as Promise<Types.UserTransaction>
  );
};

export const getAssetFullyQualifiedType = (
  tokenBridgeAddress: string, // 32 bytes
  originChain: ChainId,
  originAddress: string
): string | null => {
  // native asset
  if (originChain === CHAIN_ID_APTOS) {
    // originAddress should be of form address::module::type
    if (!isValidAptosType(originAddress)) {
      console.error("Need fully qualified address for native asset");
      return null;
    }

    return ensureHexPrefix(originAddress);
  }

  // non-native asset, derive unique address
  const wrappedAssetAddress = getForeignAssetAddress(
    tokenBridgeAddress,
    originChain,
    originAddress
  );
  return wrappedAssetAddress
    ? `${ensureHexPrefix(wrappedAssetAddress)}::coin::T`
    : null;
};

export const getForeignAssetAddress = (
  tokenBridgeAddress: string, // 32 bytes
  originChain: ChainId,
  originAddress: string
): string | null => {
  if (originChain === CHAIN_ID_APTOS) {
    return null;
  }

  // from https://github.com/aptos-labs/aptos-core/blob/25696fd266498d81d346fe86e01c330705a71465/aptos-move/framework/aptos-framework/sources/account.move#L90-L95
  let DERIVE_RESOURCE_ACCOUNT_SCHEME = Buffer.alloc(1);
  DERIVE_RESOURCE_ACCOUNT_SCHEME.writeUInt8(255);

  let chain: Buffer = Buffer.alloc(2);
  chain.writeUInt16BE(originChain);
  return sha3_256(
    Buffer.concat([
      hex(hexZeroPad(ensureHexPrefix(tokenBridgeAddress), 32)),
      chain,
      Buffer.from("::", "ascii"),
      hex(hexZeroPad(ensureHexPrefix(originAddress), 32)),
      DERIVE_RESOURCE_ACCOUNT_SCHEME,
    ])
  );
};

export const isValidAptosType = (address: string) =>
  /(0x)?[0-9a-fA-F]+::\w+::\w+/g.test(address);

export async function getFullyQualifiedTypeFromHash(
  client: AptosClient,
  tokenBridgeAddress: string,
  fullyQualifiedTypeHash: string
): Promise<string | null> {
  // get handle
  tokenBridgeAddress = ensureHexPrefix(tokenBridgeAddress);
  const state = (
    await client.getAccountResource(
      tokenBridgeAddress,
      `${tokenBridgeAddress}::state::State`
    )
  ).data as State;
  const handle = state.native_infos.handle;

  try {
    // get type info
    const typeInfo = await client.getTableItem(handle, {
      key_type: `${tokenBridgeAddress}::token_hash::TokenHash`,
      value_type: "0x1::type_info::TypeInfo",
      key: { hash: fullyQualifiedTypeHash },
    });

    if (!typeInfo) {
      return null;
    }

    // construct type
    const moduleName = Buffer.from(
      typeInfo.module_name.substring(2),
      "hex"
    ).toString("ascii");
    const structName = Buffer.from(
      typeInfo.struct_name.substring(2),
      "hex"
    ).toString("ascii");
    return `${typeInfo.account_address}::${moduleName}::${structName}`;
  } catch {
    return null;
  }
}
