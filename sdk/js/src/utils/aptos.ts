import { hexZeroPad } from "ethers/lib/utils";
import { sha3_256 } from "js-sha3";
import { ChainId, CHAIN_ID_APTOS, ensureHexPrefix, hex } from "../utils";
import { AptosAccount, AptosClient, TxnBuilderTypes, Types } from "aptos";
import { State } from "../aptos/types";

/**
 * Generate, sign, and submit a transaction calling the given entry function with the given 
 * arguments. Prevents transaction submission and throws if the transaction fails. 
 * 
 * This is separated from `generateSignAndSubmitScript` because it makes use of `AptosClient`'s 
 * `generateTransaction` which pulls ABIs from the node and uses them to encode arguments
 * automatically.
 * @param client Client used to transfer data to/from Aptos node
 * @param sender Account that will submit transaction
 * @param payload Payload containing unencoded fully qualified entry function, types, and arguments
 * @param opts Override default transaction options
 * @returns Data from transaction after is has been successfully submitted to mempool
 */
export const generateSignAndSubmitEntryFunction = (
  client: AptosClient,
  sender: AptosAccount,
  payload: Types.EntryFunctionPayload,
  opts?: Partial<Types.SubmitTransactionRequest>
): Promise<Types.UserTransaction> => {
  // overwriting `max_gas_amount` and `gas_unit_price` defaults
  // rest of defaults are defined here: https://aptos-labs.github.io/ts-sdk-doc/classes/AptosClient.html#generateTransaction
  const customOpts = Object.assign(
    {
      gas_unit_price: "100",
      max_gas_amount: "30000",
    },
    opts
  );

  return client
    .generateTransaction(sender.address(), payload, customOpts)
    .then(
      (rawTx) =>
        signAndSubmitTransaction(
          client,
          sender,
          rawTx
        ) as Promise<Types.UserTransaction>
    );
};

/**
 * Generate, sign, and submit a transaction containing given bytecode. Prevents transaction 
 * submission and throws if the transaction fails. 
 * 
 * Unlike `generateSignAndSubmitEntryFunction`, this function must construct a `RawTransaction` 
 * manually because `generateTransaction` does not have support for scripts for which there are 
 * no corresponding on-chain ABIs. Type/argument encoding is left to the caller.
 * @param client Client used to transfer data to/from Aptos node
 * @param sender Account that will submit transaction
 * @param payload Payload containing compiled bytecode and encoded types/arguments
 * @param opts Override default transaction options
 * @returns Data from transaction after is has been successfully submitted to mempool
 */
export const generateSignAndSubmitScript = async (
  client: AptosClient,
  sender: AptosAccount,
  payload: TxnBuilderTypes.TransactionPayloadScript,
  opts?: Partial<Types.SubmitTransactionRequest>
) => {
  // overwriting `max_gas_amount` and `gas_unit_price` defaults
  // rest of defaults are defined here: https://aptos-labs.github.io/ts-sdk-doc/classes/AptosClient.html#generateTransaction
  const customOpts = Object.assign(
    {
      gas_unit_price: "100",
      max_gas_amount: "30000",
    },
    opts
  );

  // create raw transaction
  const [{ sequence_number: sequenceNumber }, chainId] = await Promise.all([
    client.getAccount(sender.address()),
    client.getChainId(),
  ]);
  const rawTx = new TxnBuilderTypes.RawTransaction(
    TxnBuilderTypes.AccountAddress.fromHex(sender.address()),
    BigInt(sequenceNumber),
    payload,
    BigInt(customOpts.max_gas_amount),
    BigInt(customOpts.gas_unit_price),
    BigInt(Math.floor(Date.now() / 1000) + 10),
    new TxnBuilderTypes.ChainId(chainId)
  );
  // sign & submit transaction
  return signAndSubmitTransaction(client, sender, rawTx);
};

/**
 * Derives the fully qualified type of the asset defined by the given origin chain and address.
 * @param tokenBridgeAddress Address of token bridge (32 bytes)
 * @param originChain Chain ID of chain that original asset is from
 * @param originAddress Native address of asset; if origin chain ID is 22 (Aptos), this is the
 * asset's fully qualified type 
 * @returns The fully qualified type on Aptos for the given asset
 */
export const getAssetFullyQualifiedType = (
  tokenBridgeAddress: string,
  originChain: ChainId,
  originAddress: string
): string | null => {
  // native asset
  if (originChain === CHAIN_ID_APTOS) {
    // originAddress should be of form address::module::type
    if (!isValidAptosType(originAddress)) {
      console.error("Invalid qualified type");
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

/**
 * Derive the module address for an asset defined by the given origin chain and address.
 * @param tokenBridgeAddress Address of token bridge (32 bytes)
 * @param originChain Chain ID of chain that original asset is from
 * @param originAddress Native address of asset
 * @returns The module address for the given asset
 */
export const getForeignAssetAddress = (
  tokenBridgeAddress: string,
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

/**
 * Test if given string is a valid fully qualified type of moduleAddress::moduleName::structName.
 * @param str String to test
 * @returns Whether or not given string is a valid type
 */
export const isValidAptosType = (str: string): boolean =>
  /^(0x)?[0-9a-fA-F]+::\w+::\w+$/.test(str);

/**
 * Hashes the given type. Because fully qualified types are a concept unique to Aptos, this
 * output acts as the address on other chains.
 * @param fullyQualifiedType Fully qualified type on Aptos
 * @returns External address corresponding to given type
 */
export const getExternalAddressFromType = (
  fullyQualifiedType: string
): string => {
  // hash the type so it fits into 32 bytes
  return sha3_256(fullyQualifiedType);
};

/**
 * Given a hash, returns the fully qualified type by querying the corresponding TypeInfo.
 * @param client Client used to transfer data to/from Aptos node
 * @param tokenBridgeAddress Address of token bridge
 * @param fullyQualifiedTypeHash Hash of fully qualified type
 * @returns The fully qualified type associated with the given hash
 */
export async function getTypeFromExternalAddress(
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

/**
 * Returns module address from given fully qualified type/module address.
 * @param str FQT or module address
 * @returns Module address
 */
export const coalesceModuleAddress = (str: string): string => {
  return str.split("::")[0];
};

/**
 * Simulates given raw transaction and either returns the resulting transaction that was submitted
 * to the mempool, or throws if it fails.
 * @param client Client used to transfer data to/from Aptos node
 * @param sender Account that will submit transaction
 * @param rawTx Raw transaction to sign & submit
 * @returns Transaction data
 */
const signAndSubmitTransaction = async (
  client: AptosClient,
  sender: AptosAccount,
  rawTx: TxnBuilderTypes.RawTransaction
): Promise<Types.Transaction> => {
  // simulate transaction
  await client.simulateTransaction(sender, rawTx).then((sims) =>
    sims.forEach((tx) => {
      if (!tx.success) {
        throw new Error(
          `Transaction failed: ${tx.vm_status}\n${JSON.stringify(tx, null, 2)}`
        );
      }
    })
  );

  // sign & submit transaction
  return client
    .signTransaction(sender, rawTx)
    .then((signedTx) => client.submitTransaction(signedTx))
    .then((pendingTx) => client.waitForTransactionWithResult(pendingTx.hash));
};
