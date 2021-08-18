import { Connection, PublicKey, SystemProgram } from "@solana/web3.js";

export async function getBridgeFeeIx(
  connection: Connection,
  bridgeAddress: string,
  payerAddress: string
) {
  const bridge = await import("./core/bridge");
  const feeAccount = await bridge.fee_collector_address(bridgeAddress);
  const bridgeStatePK = new PublicKey(bridge.state_address(bridgeAddress));
  const bridgeStateAccountInfo = await connection.getAccountInfo(bridgeStatePK);
  if (bridgeStateAccountInfo?.data === undefined) {
    throw new Error("bridge state not found");
  }
  const bridgeState = bridge.parse_state(
    new Uint8Array(bridgeStateAccountInfo?.data)
  );
  const transferIx = SystemProgram.transfer({
    fromPubkey: new PublicKey(payerAddress),
    toPubkey: new PublicKey(feeAccount),
    lamports: bridgeState.config.fee,
  });
  return transferIx;
}
