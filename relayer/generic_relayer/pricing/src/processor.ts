import { Next } from "relayer-engine";
import {
  RelayProviderContractState,
  printableState,
  pullAllCurrentPricingStates,
} from "./utils/currentPricing";
import { PricingContext } from "./app";
import {
  InvariantViolation,
  calcNewViolations,
  printableViolations,
} from "./utils/stateInvariants";
import { createStateProposal } from "./utils/stateProposal";
import { checkProposedStateUpdate } from "./utils/safeguards";

export async function processProviderPriceUpdate(
  ctx: PricingContext,
  next: Next
) {
  ctx.logger.info("Starting price update process!");

  let currentState: RelayProviderContractState[] = await retrieveCurrentState(
    ctx
  );

  let currentInvariantViolations: InvariantViolation[] =
    calcInvariantViolations(ctx, currentState);

  let proposedState: RelayProviderContractState[] = await createStateProposal(
    ctx,
    currentState,
    currentInvariantViolations
  );

  let proposedInvariantViolations: InvariantViolation[] =
    calcInvariantViolations(ctx, proposedState);

  const newViolations = calcNewViolations(
    currentInvariantViolations,
    proposedInvariantViolations
  );
  if (newViolations.length > 0) {
    ctx.logger.error("New violations found!");
    ctx.logger.error(printableViolations(newViolations));
    ctx.logger.info("Exiting price update process due to new violations!");
    next();
    return;
  }

  const safeguardFlag = await checkProposedStateUpdate(ctx, currentState, çv);

  if (!safeguardFlag) {
    ctx.logger.error("Proposed state update failed safeguards!");
    ctx.logger.info("Exiting price update process due to failed safeguards!");
    next();
    return;
  }

  ctx.logger.info("Proposed state update passed all safeguards!");
  ctx.logger.info("Entering state update process...");
  const stateUpdateResult = executeStateUpdate(
    ctx,
    currentState,
    proposedState
  );

  ctx.logger.info("Exited updated state process!");
  ctx.logger.info(stateUpdateResult);
}

async function retrieveCurrentState(
  ctx: PricingContext
): Promise<RelayProviderContractState[]> {
  try {
    ctx.logger.info("Entering current state reading process...");
    const currentState = await pullAllCurrentPricingStates(ctx);
    ctx.logger.info("Successfully pulled full state!");
    for (const value of currentState) {
      ctx.logger.debug(printableState(value));
    }
    return currentState;
  } catch (e) {
    ctx.logger.error("Error pulling current state!");
    ctx.logger.error(e);
  }

  return [];
}

function calcInvariantViolations(
  ctx: PricingContext,
  currentState: RelayProviderContractState[]
): InvariantViolation[] {
  //degenerate case
  if (currentState.length <= 1) {
    return [];
  }
  try {
    ctx.logger.info("Entering invariant violation calculation process...");
    const violations = calcInvariantViolations(ctx, currentState);
    ctx.logger.info("");
    ctx.logger.debug(printableViolations(violations));

    return violations;
  } catch (e) {
    ctx.logger.error("Unhandled error during state violation calc!");
    ctx.logger.error(e);
  }

  return [];
}

async function executeStateUpdate(
  ctx: PricingContext,
  currentState: RelayProviderContractState[],
  proposedState: RelayProviderContractState[]
) {
  return true; //TODO complicated diffing & many transactions
}
