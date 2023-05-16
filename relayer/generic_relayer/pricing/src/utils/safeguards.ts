import { PricingContext } from "../app";
import { RelayProviderContractState } from "./currentPricing";

export async function checkProposedStateUpdate(
  ctx: PricingContext,
  currentState: RelayProviderContractState[],
  proposedState: RelayProviderContractState[]
): Promise<boolean> {
  //TODO
  return true;
}
