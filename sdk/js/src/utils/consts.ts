export const CHAINS = {
  unset: 0,
  solana: 1,
  ethereum: 2,
  terra: 3,
  bsc: 4,
  polygon: 5,
  avalanche: 6,
  oasis: 7,
  algorand: 8,
  aurora: 9,
  fantom: 10,
  karura: 11,
  acala: 12,
  klaytn: 13,
  ropsten: 10001,
} as const;

export type ChainName = keyof typeof CHAINS
export type ChainId = typeof CHAINS[ChainName]

/**
 *
 * All the EVM-based chain names that Wormhole supports
 */
export type EVMChainName
  = 'ethereum'
  | 'bsc'
  | 'polygon'
  | 'avalanche'
  | 'oasis'
  | 'aurora'
  | 'fantom'
  | 'karura'
  | 'acala'
  | 'klaytn'
  | 'ropsten'


// We don't specify the types of the below consts to be [[ChainId]]. This way,
// the inferred type will be a singleton (or literal) type, which is more precise and allows
// typescript to perform context-sensitive narrowing when checking against them.
// See the [[isEVMChain]] for an example.
export const CHAIN_ID_UNSET = CHAINS['unset'];
export const CHAIN_ID_SOLANA = CHAINS['solana'];
export const CHAIN_ID_ETH = CHAINS['ethereum'];
export const CHAIN_ID_TERRA = CHAINS['terra'];
export const CHAIN_ID_BSC = CHAINS['bsc'];
export const CHAIN_ID_POLYGON = CHAINS['polygon'];
export const CHAIN_ID_AVAX = CHAINS['avalanche'];
export const CHAIN_ID_OASIS = CHAINS['oasis'];
export const CHAIN_ID_ALGORAND = CHAINS['algorand'];
export const CHAIN_ID_AURORA = CHAINS['aurora'];
export const CHAIN_ID_FANTOM = CHAINS['fantom'];
export const CHAIN_ID_KARURA = CHAINS['karura'];
export const CHAIN_ID_ACALA = CHAINS['acala'];
export const CHAIN_ID_KLAYTN = CHAINS['klaytn'];
export const CHAIN_ID_ETHEREUM_ROPSTEN = CHAINS['ropsten'];

// This inverts the [[CHAINS]] object so that we can look up a chain by id
export type ChainIdToName = { -readonly [key in keyof typeof CHAINS as typeof CHAINS[key]]: key };
export const CHAIN_ID_TO_NAME: ChainIdToName = Object.entries(CHAINS).reduce((obj, [name, id]) => {
  obj[id] = name
  return obj;
}, {} as any) as ChainIdToName;

/**
 *
 * All the EVM-based chain ids that Wormhole supports
 */
export type EVMChainId = typeof CHAINS[EVMChainName]

/**
 *
 * Returns true when called with a valid chain, and narrows the type in the
 * "true" branch to [[ChainId]] or [[ChainName]] thanks to the type predicate in
 * the return type.
 *
 * A typical use-case might look like
 * ```typescript
 * foo = isChain(c) ? doSomethingWithChainId(c) : handleInvalidCase()
 * ```
 */
export function isChain(chain: number | string): chain is ChainId | ChainName {
  if (typeof chain === "number") {
    return chain in CHAIN_ID_TO_NAME
  } else {
    return chain in CHAINS
  }
}

/**
 *
 * Asserts that the given number or string is a valid chain, and throws otherwise.
 * After calling this function, the type of chain will be narrowed to
 * [[ChainId]] or [[ChainName]] thanks to the type assertion in the return type.
 *
 * A typical use-case might look like
 * ```typescript
 * // c has type 'string'
 * assertChain(c)
 * // c now has type 'ChainName'
 * ```
 */
export function assertChain(chain: number | string): asserts chain is ChainId | ChainName {
  if (!isChain(chain)) {
    if (typeof chain === "number") {
      throw Error(`Unknown chain id: ${chain}`)
    } else {
      throw Error(`Unknown chain: ${chain}`)
    }
  }
}

export function toChainId(chainName: ChainName): ChainId {
  return CHAINS[chainName]
}

export function toChainName(chainId: ChainId): ChainName {
  return CHAIN_ID_TO_NAME[chainId]
}

export function coalesceChainId(chain: ChainId | ChainName): ChainId {
  // this is written in a way that for invalid inputs (coming from vanilla
  // javascript or someone doing type casting) it will always return undefined.
  return typeof chain === "number" && isChain(chain) ? chain : toChainId(chain)
}

export function coalesceChainName(chain: ChainId | ChainName): ChainName {
  // this is written in a way that for invalid inputs (coming from vanilla
  // javascript or someone doing type casting) it will always return undefined.
  return toChainName(coalesceChainId(chain))
}

/**
 *
 * Returns true when called with an [[EVMChainId]] or [[EVMChainName]], and false otherwise.
 * Importantly, after running this check, the chain's type will be narrowed to
 * either the EVM subset, or the non-EVM subset thanks to the type predicate in
 * the return type.
 */
export function isEVMChain(chain: ChainId | ChainName): chain is EVMChainId | EVMChainName {
  let chainId = coalesceChainId(chain)
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
    chainId === CHAIN_ID_ETHEREUM_ROPSTEN
  ) {
    return isEVM(chainId)
  } else {
    return notEVM(chainId)
  }
}

/**
 *
 * Asserts that the given chain id or chain name is an EVM chain, and throws otherwise.
 * After calling this function, the type of chain will be narrowed to
 * [[EVMChainId]] or [[EVMChainName]] thanks to the type assertion in the return type.
 *
 */
export function assertEVMChain(chain: ChainId | ChainName): asserts chain is EVMChainId | EVMChainName {
  if (!isEVMChain(chain)) {
    throw Error(`Expected an EVM chain, but ${chain} is not`)
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
 * Returns true when called with an [[EVMChainId]] or [[EVMChainName]], and fails to compile
 * otherwise
 */
function isEVM(_: EVMChainId | EVMChainName): true {
  return true
}

/**
 *
 * Returns false when called with a non-[[EVMChainId]] and non-[[EVMChainName]]
 * argument, and fails to compile otherwise
 */
function notEVM<T>(_: T extends EVMChainId | EVMChainName ? never : T): false {
  return false
}
