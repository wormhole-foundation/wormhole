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
  celo: 14,
  near: 15,
  moonbeam: 16,
  neon: 17,
  terra2: 18,
  injective: 19,
  osmosis: 20,
  sui: 21,
  aptos: 22,
  ropsten: 10001,
} as const;

export type ChainName = keyof typeof CHAINS;
export type ChainId = typeof CHAINS[ChainName];

/**
 *
 * All the EVM-based chain names that Wormhole supports
 */
export type EVMChainName =
  | "ethereum"
  | "bsc"
  | "polygon"
  | "avalanche"
  | "oasis"
  | "aurora"
  | "fantom"
  | "karura"
  | "acala"
  | "klaytn"
  | "celo"
  | "moonbeam"
  | "neon"
  | "ropsten";

export type TerraChainName = "terra" | "terra2";

export type Contracts = {
  core: string | undefined;
  token_bridge: string | undefined;
  nft_bridge: string | undefined;
};

export type ChainContracts = {
  [chain in ChainName]: Contracts;
};

const MAINNET = {
  unset: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  solana: {
    core: "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth",
    token_bridge: "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb",
    nft_bridge: "WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD",
  },
  ethereum: {
    core: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B",
    token_bridge: "0x3ee18B2214AFF97000D974cf647E7C347E8fa585",
    nft_bridge: "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE",
  },
  terra: {
    core: "terra1dq03ugtd40zu9hcgdzrsq6z2z4hwhc9tqk2uy5",
    token_bridge: "terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf",
    nft_bridge: undefined,
  },
  bsc: {
    core: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B",
    token_bridge: "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7",
    nft_bridge: "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE",
  },
  polygon: {
    core: "0x7A4B5a56256163F07b2C80A7cA55aBE66c4ec4d7",
    token_bridge: "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE",
    nft_bridge: "0x90BBd86a6Fe93D3bc3ed6335935447E75fAb7fCf",
  },
  avalanche: {
    core: "0x54a8e5f9c4CbA08F9943965859F6c34eAF03E26c",
    token_bridge: "0x0e082F06FF657D94310cB8cE8B0D9a04541d8052",
    nft_bridge: "0xf7B6737Ca9c4e08aE573F75A97B73D7a813f5De5",
  },
  oasis: {
    core: "0xfE8cD454b4A1CA468B57D79c0cc77Ef5B6f64585",
    token_bridge: "0x5848C791e09901b40A9Ef749f2a6735b418d7564",
    nft_bridge: "0x04952D522Ff217f40B5Ef3cbF659EcA7b952a6c1",
  },
  algorand: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  aurora: {
    core: "0xa321448d90d4e5b0A732867c18eA198e75CAC48E",
    token_bridge: "0x51b5123a7b0F9b2bA265f9c4C8de7D78D52f510F",
    nft_bridge: "0x6dcC0484472523ed9Cdc017F711Bcbf909789284",
  },
  fantom: {
    core: "0x126783A6Cb203a3E35344528B26ca3a0489a1485",
    token_bridge: "0x7C9Fc5741288cDFdD83CeB07f3ea7e22618D79D2",
    nft_bridge: "0xA9c7119aBDa80d4a4E0C06C8F4d8cF5893234535",
  },
  karura: {
    core: "0xa321448d90d4e5b0A732867c18eA198e75CAC48E",
    token_bridge: "0xae9d7fe007b3327AA64A32824Aaac52C42a6E624",
    nft_bridge: "0xb91e3638F82A1fACb28690b37e3aAE45d2c33808",
  },
  acala: {
    core: "0xa321448d90d4e5b0A732867c18eA198e75CAC48E",
    token_bridge: "0xae9d7fe007b3327AA64A32824Aaac52C42a6E624",
    nft_bridge: "0xb91e3638F82A1fACb28690b37e3aAE45d2c33808",
  },
  klaytn: {
    core: "0x0C21603c4f3a6387e241c0091A7EA39E43E90bb7",
    token_bridge: "0x5b08ac39EAED75c0439FC750d9FE7E1F9dD0193F",
    nft_bridge: "0x3c3c561757BAa0b78c5C025CdEAa4ee24C1dFfEf",
  },
  celo: {
    core: "0xa321448d90d4e5b0A732867c18eA198e75CAC48E",
    token_bridge: "0x796Dff6D74F3E27060B71255Fe517BFb23C93eed",
    nft_bridge: "0xA6A377d75ca5c9052c9a77ED1e865Cc25Bd97bf3",
  },
  near: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  injective: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  osmosis: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  aptos: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  sui: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  moonbeam: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  neon: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  terra2: {
    core: "terra12mrnzvhx3rpej6843uge2yyfppfyd3u9c3uq223q8sl48huz9juqffcnhp",
    token_bridge:
      "terra153366q50k7t8nn7gec00hg66crnhkdggpgdtaxltaq6xrutkkz3s992fw9",
    nft_bridge: undefined,
  },
  ropsten: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
};

const TESTNET = {
  unset: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  solana: {
    core: "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5",
    token_bridge: "DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe",
    nft_bridge: "2rHhojZ7hpu1zA91nvZmT8TqWWvMcKmmNBCr2mKTtMq4",
  },
  terra: {
    core: "terra1pd65m0q9tl3v8znnz5f5ltsfegyzah7g42cx5v",
    token_bridge: "terra1pseddrv0yfsn76u4zxrjmtf45kdlmalswdv39a",
    nft_bridge: undefined,
  },
  ethereum: {
    core: "0x706abc4E45D419950511e474C7B9Ed348A4a716c",
    token_bridge: "0xF890982f9310df57d00f659cf4fd87e65adEd8d7",
    nft_bridge: "0xD8E4C2DbDd2e2bd8F1336EA691dBFF6952B1a6eB",
  },
  bsc: {
    core: "0x68605AD7b15c732a30b1BbC62BE8F2A509D74b4D",
    token_bridge: "0x9dcF9D205C9De35334D646BeE44b2D2859712A09",
    nft_bridge: "0xcD16E5613EF35599dc82B24Cb45B5A93D779f1EE",
  },
  polygon: {
    core: "0x0CBE91CF822c73C2315FB05100C2F714765d5c20",
    token_bridge: "0x377D55a7928c046E18eEbb61977e714d2a76472a",
    nft_bridge: "0x51a02d0dcb5e52F5b92bdAA38FA013C91c7309A9",
  },
  avalanche: {
    core: "0x7bbcE28e64B3F8b84d876Ab298393c38ad7aac4C",
    token_bridge: "0x61E44E506Ca5659E6c0bba9b678586fA2d729756",
    nft_bridge: "0xD601BAf2EEE3C028344471684F6b27E789D9075D",
  },
  oasis: {
    core: "0xc1C338397ffA53a2Eb12A7038b4eeb34791F8aCb",
    token_bridge: "0x88d8004A9BdbfD9D28090A02010C19897a29605c",
    nft_bridge: "0xC5c25B41AB0b797571620F5204Afa116A44c0ebA",
  },
  algorand: {
    core: "86525623",
    token_bridge: "86525641",
    nft_bridge: undefined,
  },
  aurora: {
    core: "0xBd07292de7b505a4E803CEe286184f7Acf908F5e",
    token_bridge: "0xD05eD3ad637b890D68a854d607eEAF11aF456fba",
    nft_bridge: "0x8F399607E9BA2405D87F5f3e1B78D950b44b2e24",
  },
  fantom: {
    core: "0x1BB3B4119b7BA9dfad76B0545fb3F531383c3bB7",
    token_bridge: "0x599CEa2204B4FaECd584Ab1F2b6aCA137a0afbE8",
    nft_bridge: "0x63eD9318628D26BdCB15df58B53BB27231D1B227",
  },
  karura: {
    core: "0xE4eacc10990ba3308DdCC72d985f2a27D20c7d03",
    token_bridge: "0xd11De1f930eA1F7Dd0290Fe3a2e35b9C91AEFb37",
    nft_bridge: "0x0A693c2D594292B6Eb89Cb50EFe4B0b63Dd2760D",
  },
  acala: {
    core: "0x4377B49d559c0a9466477195C6AdC3D433e265c0",
    token_bridge: "0xebA00cbe08992EdD08ed7793E07ad6063c807004",
    nft_bridge: "0x96f1335e0AcAB3cfd9899B30b2374e25a2148a6E",
  },
  klaytn: {
    core: "0x1830CC6eE66c84D2F177B94D544967c774E624cA",
    token_bridge: "0xC7A13BE098720840dEa132D860fDfa030884b09A",
    nft_bridge: "0x94c994fC51c13101062958b567e743f1a04432dE",
  },
  celo: {
    core: "0x88505117CA88e7dd2eC6EA1E13f0948db2D50D56",
    token_bridge: "0x05ca6037eC51F8b712eD2E6Fa72219FEaE74E153",
    nft_bridge: "0xaCD8190F647a31E56A656748bC30F69259f245Db",
  },
  near: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  injective: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  osmosis: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  aptos: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  sui: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  moonbeam: {
    core: "0xa5B7D85a8f27dd7907dc8FdC21FA5657D5E2F901",
    token_bridge: "0xbc976D4b9D57E57c3cA52e1Fd136C45FF7955A96",
    nft_bridge: "0x98A0F4B96972b32Fcb3BD03cAeB66A44a6aB9Edb",
  },
  neon: {
    core: "0xE4eacc10990ba3308DdCC72d985f2a27D20c7d03",
    token_bridge: "0xd11De1f930eA1F7Dd0290Fe3a2e35b9C91AEFb37",
    nft_bridge: "0xa52Da3B1ffd258a2fFB7719a6aeE24095eEE24E2",
  },
  terra2: {
    core: "terra19nv3xr5lrmmr7egvrk2kqgw4kcn43xrtd5g0mpgwwvhetusk4k7s66jyv0",
    token_bridge:
      "terra1c02vds4uhgtrmcw7ldlg75zumdqxr8hwf7npseuf2h58jzhpgjxsgmwkvk",
    nft_bridge: undefined,
  },
  ropsten: {
    core: "0x210c5F5e2AF958B4defFe715Dc621b7a3BA888c5",
    token_bridge: "0xF174F9A837536C449321df1Ca093Bb96948D5386",
    nft_bridge: "0x2b048Da40f69c8dc386a56705915f8E966fe1eba",
  },
};

const DEVNET = {
  unset: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  solana: {
    core: "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o",
    token_bridge: "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE",
    nft_bridge: "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA",
  },
  terra: {
    core: "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5",
    token_bridge: "terra10pyejy66429refv3g35g2t7am0was7ya7kz2a4",
    nft_bridge: undefined,
  },
  ethereum: {
    core: "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550",
    token_bridge: "0x0290FB167208Af455bB137780163b7B7a9a10C16",
    nft_bridge: "0x26b4afb60d6c903165150c6f0aa14f8016be4aec",
  },
  bsc: {
    core: "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550",
    token_bridge: "0x0290FB167208Af455bB137780163b7B7a9a10C16",
    nft_bridge: "0x26b4afb60d6c903165150c6f0aa14f8016be4aec",
  },
  polygon: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  avalanche: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  oasis: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  algorand: {
    core: "4",
    token_bridge: "6",
    nft_bridge: undefined,
  },
  aurora: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  fantom: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  karura: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  acala: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  klaytn: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  celo: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  near: {
    core: "wormhole.test.near",
    token_bridge: "token.test.near",
    nft_bridge: undefined,
  },
  injective: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  osmosis: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  aptos: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  sui: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  moonbeam: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  neon: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  terra2: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  ropsten: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
};

/**
 *
 * If you get a type error here, it means that a chain you just added does not
 * have an entry in TESTNET.
 * This is implemented as an ad-hoc type assertion instead of a type annotation
 * on TESTNET so that e.g.
 *
 * ```typescript
 * TESTNET['solana'].core
 * ```
 * has type 'string' instead of 'string | undefined'.
 *
 * (Do not delete this declaration!)
 */
const isTestnetContracts: ChainContracts = TESTNET;

/**
 *
 * See [[isTestnetContracts]]
 */
const isMainnetContracts: ChainContracts = MAINNET;

/**
 *
 * See [[isTestnetContracts]]
 */
const isDevnetContracts: ChainContracts = DEVNET;

/**
 *
 * Contracts addresses on testnet and mainnet
 */
export const CONTRACTS = { MAINNET, TESTNET, DEVNET };

// We don't specify the types of the below consts to be [[ChainId]]. This way,
// the inferred type will be a singleton (or literal) type, which is more precise and allows
// typescript to perform context-sensitive narrowing when checking against them.
// See the [[isEVMChain]] for an example.
export const CHAIN_ID_UNSET = CHAINS["unset"];
export const CHAIN_ID_SOLANA = CHAINS["solana"];
export const CHAIN_ID_ETH = CHAINS["ethereum"];
export const CHAIN_ID_TERRA = CHAINS["terra"];
export const CHAIN_ID_BSC = CHAINS["bsc"];
export const CHAIN_ID_POLYGON = CHAINS["polygon"];
export const CHAIN_ID_AVAX = CHAINS["avalanche"];
export const CHAIN_ID_OASIS = CHAINS["oasis"];
export const CHAIN_ID_ALGORAND = CHAINS["algorand"];
export const CHAIN_ID_AURORA = CHAINS["aurora"];
export const CHAIN_ID_FANTOM = CHAINS["fantom"];
export const CHAIN_ID_KARURA = CHAINS["karura"];
export const CHAIN_ID_ACALA = CHAINS["acala"];
export const CHAIN_ID_KLAYTN = CHAINS["klaytn"];
export const CHAIN_ID_CELO = CHAINS["celo"];
export const CHAIN_ID_NEAR = CHAINS["near"];
export const CHAIN_ID_MOONBEAM = CHAINS["moonbeam"];
export const CHAIN_ID_NEON = CHAINS["neon"];
export const CHAIN_ID_TERRA2 = CHAINS["terra2"];
export const CHAIN_ID_INJECTIVE = CHAINS["injective"];
export const CHAIN_ID_OSMOSIS = CHAINS["osmosis"];
export const CHAIN_ID_SUI = CHAINS["sui"];
export const CHAIN_ID_APTOS = CHAINS["aptos"];
export const CHAIN_ID_ETHEREUM_ROPSTEN = CHAINS["ropsten"];

// This inverts the [[CHAINS]] object so that we can look up a chain by id
export type ChainIdToName = {
  -readonly [key in keyof typeof CHAINS as typeof CHAINS[key]]: key;
};
export const CHAIN_ID_TO_NAME: ChainIdToName = Object.entries(CHAINS).reduce(
  (obj, [name, id]) => {
    obj[id] = name;
    return obj;
  },
  {} as any
) as ChainIdToName;

/**
 *
 * All the EVM-based chain ids that Wormhole supports
 */
export type EVMChainId = typeof CHAINS[EVMChainName];

export type TerraChainId = typeof CHAINS[TerraChainName];

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
    return chain in CHAIN_ID_TO_NAME;
  } else {
    return chain in CHAINS;
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
export function assertChain(
  chain: number | string
): asserts chain is ChainId | ChainName {
  if (!isChain(chain)) {
    if (typeof chain === "number") {
      throw Error(`Unknown chain id: ${chain}`);
    } else {
      throw Error(`Unknown chain: ${chain}`);
    }
  }
}

export function toChainId(chainName: ChainName): ChainId {
  return CHAINS[chainName];
}

export function toChainName(chainId: ChainId): ChainName {
  return CHAIN_ID_TO_NAME[chainId];
}

export function coalesceChainId(chain: ChainId | ChainName): ChainId {
  // this is written in a way that for invalid inputs (coming from vanilla
  // javascript or someone doing type casting) it will always return undefined.
  return typeof chain === "number" && isChain(chain) ? chain : toChainId(chain);
}

export function coalesceChainName(chain: ChainId | ChainName): ChainName {
  // this is written in a way that for invalid inputs (coming from vanilla
  // javascript or someone doing type casting) it will always return undefined.
  return toChainName(coalesceChainId(chain));
}

/**
 *
 * Returns true when called with an [[EVMChainId]] or [[EVMChainName]], and false otherwise.
 * Importantly, after running this check, the chain's type will be narrowed to
 * either the EVM subset, or the non-EVM subset thanks to the type predicate in
 * the return type.
 */
export function isEVMChain(
  chain: ChainId | ChainName
): chain is EVMChainId | EVMChainName {
  let chainId = coalesceChainId(chain);
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
    chainId === CHAIN_ID_MOONBEAM ||
    chainId === CHAIN_ID_NEON ||
    chainId === CHAIN_ID_ETHEREUM_ROPSTEN
  ) {
    return isEVM(chainId);
  } else {
    return notEVM(chainId);
  }
}

export function isTerraChain(
  chain: ChainId | ChainName
): chain is TerraChainId | TerraChainName {
  const chainId = coalesceChainId(chain);
  return chainId === CHAIN_ID_TERRA || chainId === CHAIN_ID_TERRA2;
}

/**
 *
 * Asserts that the given chain id or chain name is an EVM chain, and throws otherwise.
 * After calling this function, the type of chain will be narrowed to
 * [[EVMChainId]] or [[EVMChainName]] thanks to the type assertion in the return type.
 *
 */
export function assertEVMChain(
  chain: ChainId | ChainName
): asserts chain is EVMChainId | EVMChainName {
  if (!isEVMChain(chain)) {
    throw Error(`Expected an EVM chain, but ${chain} is not`);
  }
}

export const WSOL_ADDRESS = "So11111111111111111111111111111111111111112";
export const WSOL_DECIMALS = 9;
export const MAX_VAA_DECIMALS = 8;

// TODO: will this work for terra2?
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
  return true;
}

/**
 *
 * Returns false when called with a non-[[EVMChainId]] and non-[[EVMChainName]]
 * argument, and fails to compile otherwise
 */
function notEVM<T>(_: T extends EVMChainId | EVMChainName ? never : T): false {
  return false;
}

// This just serves as a type assertion to ensure that [[EVMChainName]] is a
// subset of [[ChainName]], since typescript provides no built-in way to express
// this.
function evm_chain_subset(e: EVMChainName): ChainName {
  // will fail to compile if 'e' can't be typed as a [[ChainName]]
  return e;
}
