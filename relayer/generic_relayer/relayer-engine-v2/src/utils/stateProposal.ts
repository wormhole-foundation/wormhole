import { PricingContext } from "../app";
import { DeliveryProviderContractState } from "./currentPricing";
import { InvariantViolation } from "./stateInvariants";

export async function createStateProposal(
  ctx: PricingContext,
  currentState: DeliveryProviderContractState[],
  currentInvariantViolations: InvariantViolation[]
): Promise<DeliveryProviderContractState[]> {
  return loadUpdatedProposalFromFileSystem();
}

function loadUpdatedProposalFromFileSystem(): DeliveryProviderContractState[] {
  //TODO load from file system
  return [];
}

function createDynamicProposal(): DeliveryProviderContractState[] {
  //TODO Do this however it should be done according to monitoring logic
  return [];
}
