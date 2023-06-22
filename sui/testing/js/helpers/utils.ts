import { JsonRpcProvider } from "@mysten/sui.js";

export async function getPackageId(
  provider: JsonRpcProvider,
  stateId: string
): Promise<string> {
  const state = await provider
    .getObject({
      id: stateId,
      options: {
        showContent: true,
      },
    })
    .then((result) => {
      if (result.data?.content?.dataType == "moveObject") {
        return result.data.content.fields;
      }

      throw new Error("not move object");
    });

  if ("upgrade_cap" in state) {
    return state.upgrade_cap.fields.package;
  }

  throw new Error("upgrade_cap not found");
}
