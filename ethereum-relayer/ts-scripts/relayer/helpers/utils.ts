import { ContractReceipt, ContractTransaction } from "ethers";

export function wait(tx: ContractTransaction): Promise<ContractReceipt> {
  return tx.wait();
}
