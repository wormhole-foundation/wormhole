import { HDKey } from "@scure/bip32";
import { mnemonicToSeed } from "@scure/bip39";
import { createClient } from "@stacks/blockchain-api-client";
import type { StacksNetworkName } from "@stacks/network";
import { privateKeyToAddress } from "@stacks/transactions";
import { DerivationType, deriveAccount, deriveWalletKeys, getStxAddress } from "@stacks/wallet-sdk";

export async function waitForTransactionSuccess(
  stacksApiUrl: string,
  txid: string,
  timeoutMs = 30_000
): Promise<void> {
  const api = createClient({ baseUrl: stacksApiUrl });
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    try {
      const tx = await api.GET("/extended/v1/tx/{tx_id}", {
        params: { path: { tx_id: txid } },
      });

      if (tx.data?.tx_status === "success") {
        return;
      } else if (
        tx.data?.tx_status === "abort_by_response" ||
        tx.data?.tx_status === "abort_by_post_condition"
      ) {
        const result =
          (tx.data as any).tx_result?.repr || "No result available";
        const contractId = (tx.data as any).smart_contract?.contract_id || "";

        console.error(
          `Transaction ${txid} failed with status: ${tx.data.tx_status}`
        );
        console.error(`Result: ${result}`);
        console.error(`Contract: ${contractId}`);
        console.error(
          `Full transaction details:`,
          JSON.stringify(tx.data, null, 2)
        );

        // Provide more context about the failure
        if (result === "(err none)" && tx.data.tx_type === "smart_contract") {
          console.error(
            `Contract deployment failed - this might be due to dependency issues or contract execution errors`
          );
        }

        throw new Error(
          `Transaction ${txid} failed: ${tx.data.tx_status} - Result: ${result}`
        );
      }

      await sleep(1000);
    } catch (error) {
      if (error instanceof Error && error.message.includes("failed:")) {
        throw error;
      }
      // Continue polling if it's just a network error or tx not found yet
      await sleep(1000);
    }
  }

  throw new Error(
    `Timeout waiting for transaction ${txid} to succeed after ${timeoutMs}ms`
  );
}

export async function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function getKeys(network: StacksNetworkName, mnemonic?: string, privateKey?: string): Promise<{privateKey: string, address: string}> {
  if(!!mnemonic) {
    return mnemonicToKeys(mnemonic, network)
  } else if(!!privateKey) {
    return {
      privateKey,
      address: privateKeyToAddress(privateKey, network)
    }
  } else {
    throw new Error("One of mnemonic or privateKey must be set")
  }
}

async function mnemonicToKeys(mnemonic: string, network: StacksNetworkName): Promise<{privateKey: string, address: string}> {
  const deployerAccountIndex = Number(process.env.DEPLOYER_ACCOUNT_INDEX ?? 0)
  const rootPrivateKey = await mnemonicToSeed(mnemonic)
  const rootNode1 = HDKey.fromMasterSeed(rootPrivateKey)
  const derived = await deriveWalletKeys(rootNode1 as any)
  const rootNode = HDKey.fromExtendedKey(derived.rootKey)
  console.log(`using index ${deployerAccountIndex}`)
  const account = deriveAccount({
    rootNode: rootNode as any,
    index: deployerAccountIndex,
    salt: derived.salt,
    stxDerivationType: DerivationType.Wallet,
  })

  const address = getStxAddress({ account, network })
  return {
    privateKey: account.stxPrivateKey,
    address
  }
}
