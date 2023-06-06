import { PricingContext } from "../app";
import { DeliveryProviderContractState } from "./currentPricing";

export async function checkProposedStateUpdate(
  ctx: PricingContext,
  currentState: DeliveryProviderContractState[],
  proposedState: DeliveryProviderContractState[]
): Promise<boolean> {
  //TODO
  return true;
}
