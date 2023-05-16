import { PricingContext } from "../app";
import { RelayProviderContractState } from "./currentPricing";
import { InvariantViolation } from "./stateInvariants";

export async function createStateProposal(
  ctx: PricingContext,
  currentState: RelayProviderContractState[],
  currentInvariantViolations: InvariantViolation[]
): Promise<RelayProviderContractState[]> {
  return loadUpdatedProposalFromFileSystem();
}

function loadUpdatedProposalFromFileSystem(): RelayProviderContractState[] {
  //TODO load from file system
  return [];
}

function createDynamicProposal(): RelayProviderContractState[] {
  //TODO Do this however it should be done according to monitoring logic
  return [];
}
