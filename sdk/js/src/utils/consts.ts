export type ChainId =
  | 1
  | 2
  | 3
  | 4
  | 5
  | 6
  | 7
  | 8
  | 9
  | 10
  | 11
  | 12
  | 13
  | 14
  | 15
  | 10001;

// We don't specify the types of the below consts to be [[ChainId]]. This way,
// the inferred type will be a singleton (or literal) type, which is more precise and allows
// typescript to perform context-sensitive narrowing when checking against them.
// See the [[isEVMChain]] for an example.

export const CHAIN_ID_SOLANA = 1;
export const CHAIN_ID_ETH = 2;
export const CHAIN_ID_TERRA = 3;
export const CHAIN_ID_BSC = 4;
export const CHAIN_ID_POLYGON = 5;
export const CHAIN_ID_AVAX = 6;
export const CHAIN_ID_OASIS = 7;
export const CHAIN_ID_ALGORAND = 8;
export const CHAIN_ID_AURORA = 9;
export const CHAIN_ID_FANTOM = 10;
export const CHAIN_ID_KARURA = 11;
export const CHAIN_ID_ACALA = 12;
export const CHAIN_ID_KLAYTN = 13;
export const CHAIN_ID_CELO = 14;
export const CHAIN_ID_NEAR = 15;
export const CHAIN_ID_ETHEREUM_ROPSTEN = 10001;

/**
 * EVM-based chains behave in much the same way for most intents and purposes,
 * so it's useful to define their own type.
 */
export type EVMChainId =
  | typeof CHAIN_ID_ETH
  | typeof CHAIN_ID_BSC
  | typeof CHAIN_ID_POLYGON
  | typeof CHAIN_ID_AVAX
  | typeof CHAIN_ID_OASIS
  | typeof CHAIN_ID_AURORA
  | typeof CHAIN_ID_FANTOM
  | typeof CHAIN_ID_KARURA
  | typeof CHAIN_ID_ACALA
  | typeof CHAIN_ID_KLAYTN
  | typeof CHAIN_ID_CELO
  | typeof CHAIN_ID_ETHEREUM_ROPSTEN;

/**
 *
 * Returns true when called with an [[EVMChainId]], and false otherwise.
 * Importantly, after running this check, the chainId's type will be narrowed to
 * either the EVM subset, or the non-EVM subset thanks to the type predicate in
 * the return type.
 */
export function isEVMChain(chainId: ChainId): chainId is EVMChainId {
  if (
    chainId === CHAIN_ID_ETH ||
    chainId === CHAIN_ID_BSC ||
    chainId === CHAIN_ID_AVAX ||
    chainId === CHAIN_ID_POLYGON ||
    chainId === CHAIN_ID_OASIS ||
    chainId === CHAIN_ID_AURORA ||
    chainId === CHAIN_ID_FANTOM ||
    chainId === CHAIN_ID_KARURA ||
    chainId === CHAIN_ID_ACALA ||
    chainId === CHAIN_ID_KLAYTN ||
    chainId === CHAIN_ID_CELO ||
    chainId === CHAIN_ID_ETHEREUM_ROPSTEN
  ) {
    return isEVM(chainId);
  } else {
    return notEVM(chainId);
  }
}

export const WSOL_ADDRESS = "So11111111111111111111111111111111111111112";
export const WSOL_DECIMALS = 9;
export const MAX_VAA_DECIMALS = 8;

export const TERRA_REDEEMED_CHECK_WALLET_ADDRESS =
  "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v";

////////////////////////////////////////////////////////////////////////////////
// Utilities

/**
 * The [[isEVM]] and [[notEVM]] functions improve type-safety in [[isEVMChain]].
 *
 * As it turns out, typescript type predicates are unsound on their own,
 * allowing us to write something like this:
 *
 * ```typescript
 * function unsafeCoerce(n: number): n is 1 {
 *   return true
 * }
 * ```
 *
 * which is completely bogus. This happens presumably because the typescript
 * authors think of the type predicate mechanism as an escape hatch mechanism.
 * We want a more principled function though, that keeps us honest.
 *
 * in [[isEVMChain]], checking that disjunctive boolean expression actually
 * refines the type of chainId in both branches. In the "true" branch,
 * the type of chainId is narrowed to exactly the EVM chains, so calling
 * [[isEVM]] on it will typecheck, and similarly the "false" branch for the negation.
 * However, if we extend the [[EVMChainId]] type with a new EVM chain, this
 * function will no longer compile until the condition is extended.
 */

/**
 *
 * Returns true when called with an [[EVMChainId]], and fails to compile
 * otherwise
 */
function isEVM(_: EVMChainId): true {
  return true;
}

/**
 *
 * Returns false when called with an non-[[EVMChainId]], and fails to compile
 * otherwise
 */
function notEVM<T>(_: T extends EVMChainId ? never : T): false {
  return false;
}
