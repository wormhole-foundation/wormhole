import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { CosmwasmWasmV1Query } from "@injectivelabs/core-proto-ts";

export const parseSmartContractStateResponse = ({
  data,
}: CosmwasmWasmV1Query.QuerySmartContractStateResponse) =>
  JSON.parse(Buffer.from(data).toString());

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
