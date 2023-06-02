import { ChainId } from "@certusone/wormhole-sdk";
import { DeliveryProviderContractState, printFixed } from "./currentPricing";

export type InvariantViolation = {
  invariant: HardInvariant | SoftInvariant;
  sourceChain?: ChainId;
  sourceValue?: string;
  targetChain?: ChainId;
  targetValue?: string;
};

export enum HardInvariant {}

export enum SoftInvariant {}

export function checkAllInvariants(
  contractStates: DeliveryProviderContractState[]
): InvariantViolation[] {
  return [].concat(
    checkHardInvariants(contractStates),
    checkSoftInvariants(contractStates)
  );
}

//Hard invariants are defined as invariants which are disallowed by the Wormhole relayer protocol,
//and thus should never ever be violated.
export function checkHardInvariants(
  contractStates: DeliveryProviderContractState[]
) {
  return [];
}

//Soft invariants are defined as invariants which are not disallowed by the Wormhole relayer protocol,
//and can be modified by the relay provider. This included things like minimums, maximums, and bi-directional relationships.
export function checkSoftInvariants(
  contractStates: DeliveryProviderContractState[]
) {
  return [];
}

export function printableViolations(violations: InvariantViolation[]) {
  let output = "";
  for (const value of violations) {
    output += printableViolation(value);
    output += "\n";
  }
  return output;
}

export function printableViolation(violation: InvariantViolation) {
  let output = "";
  output += printFixed(
    typeof violation.invariant == HardInvariant ? "Hard" : "Soft",
    violation.invariant.toString()
  );
  if (violation.sourceChain) {
    output += printFixed("Source Chain", violation.sourceChain.toString());
  }
  if (violation.sourceValue) {
    output += printFixed("Source Value", violation.sourceValue.toString());
  }
  if (violation.targetChain) {
    output += printFixed("Target Chain", violation.targetChain.toString());
  }
  if (violation.targetValue) {
    output += printFixed("Target Value", violation.targetValue.toString());
  }

  return output;
}

export function calcNewViolations(
  currentViolations: InvariantViolation[],
  proposedViolations: InvariantViolation[]
): InvariantViolation[] {
  let newViolations: InvariantViolation[] = [];
  for (const value of proposedViolations) {
    let found = false;
    for (const value2 of currentViolations) {
      if (invariantViolationEquals(value, value2)) {
        found = true;
        break;
      }
    }
    if (!found) {
      newViolations.push(value);
    }
  }
  return newViolations;
}

//Custom equality function for InvariantViolation objects
export function invariantViolationEquals(
  a: InvariantViolation,
  b: InvariantViolation
) {
  return (
    a.invariant == b.invariant &&
    a.sourceChain == b.sourceChain &&
    a.sourceValue == b.sourceValue &&
    a.targetChain == b.targetChain &&
    a.targetValue == b.targetValue
  );
}
