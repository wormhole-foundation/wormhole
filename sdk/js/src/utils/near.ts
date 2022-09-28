import nearAPI from "near-api-js";
import BN from "bn.js";
import { Provider } from "near-api-js/lib/providers";
import { CodeResult } from "near-api-js/lib/providers/provider";

export function logNearGas(result: any, comment: string) {
  const { totalGasBurned, totalTokensBurned } = result.receipts_outcome.reduce(
    (acc: any, receipt: any) => {
      acc.totalGasBurned += receipt.outcome.gas_burnt;
      acc.totalTokensBurned += nearAPI.utils.format.formatNearAmount(
        receipt.outcome.tokens_burnt
      );
      return acc;
    },
    {
      totalGasBurned: result.transaction_outcome.outcome.gas_burnt,
      totalTokensBurned: nearAPI.utils.format.formatNearAmount(
        result.transaction_outcome.outcome.tokens_burnt
      ),
    }
  );
  console.log(
    comment,
    "totalGasBurned",
    totalGasBurned,
    "totalTokensBurned",
    totalTokensBurned
  );
}

export async function hashAccount(
  provider: Provider,
  tokenBridge: string,
  account: string
): Promise<{ isRegistered: boolean; accountHash: string }> {
  // Near can have account names up to 64 bytes, but wormhole only supports 32
  // As a result, we have to hash our account names with sha256
  const [isRegistered, accountHash] = await callFunctionNear(
    provider,
    tokenBridge,
    "hash_account",
    { account }
  );
  return {
    isRegistered,
    accountHash,
  };
}

export async function hashLookup(
  provider: Provider,
  tokenBridge: string,
  hash: string
): Promise<{ found: boolean; value: string }> {
  const [found, value] = await callFunctionNear(
    provider,
    tokenBridge,
    "hash_lookup",
    {
      hash,
    }
  );
  return {
    found,
    value,
  };
}

export function registerAccount(account: string, tokenBridge: string) {
  return {
    contractId: tokenBridge,
    methodName: "register_account",
    args: { account },
    gas: new BN("100000000000000"),
    attachedDeposit: new BN("2000000000000000000000"),
  };
}

export async function callFunctionNear(
  provider: Provider,
  accountId: string,
  methodName: string,
  args?: any
) {
  const response = await provider.query<CodeResult>({
    request_type: "call_function",
    account_id: accountId,
    method_name: methodName,
    args_base64: args
      ? Buffer.from(JSON.stringify(args)).toString("base64")
      : "",
    finality: "final",
  });
  return JSON.parse(Buffer.from(response.result).toString());
}
