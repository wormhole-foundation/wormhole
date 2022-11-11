import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { QuerySmartContractStateResponse } from "@injectivelabs/chain-api/cosmwasm/wasm/v1/query_pb";

export const parseSmartContractStateResponse: any = ({
  data,
}: QuerySmartContractStateResponse.AsObject) =>
  JSON.parse(
    Buffer.from(
      typeof data === "string" ? data : Buffer.from(data).toString(),
      "base64"
    ).toString()
  );

export const queryExternalIdInjective = async (
  client: ChainGrpcWasmApi,
  tokenBridgeAddress: string,
  externalTokenId: string
): Promise<string | null> => {
  try {
    const response = await client.fetchSmartContractState(
      tokenBridgeAddress,
      Buffer.from(
        JSON.stringify({
          external_id: {
            external_id: Buffer.from(externalTokenId, "hex").toString("base64"),
          },
        })
      ).toString("base64")
    );
    const parsedResponse = parseSmartContractStateResponse(response);
    const denomOrAddress: string | undefined =
      parsedResponse.token_id.Bank?.denom ||
      parsedResponse.token_id.Contract?.NativeCW20?.contract_address ||
      parsedResponse.token_id.Contract?.ForeignToken?.foreign_address;
    return denomOrAddress || null;
  } catch {
    return null;
  }
};
