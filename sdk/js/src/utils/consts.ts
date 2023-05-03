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
  arbitrum: 23,
  optimism: 24,
  gnosis: 25,
  pythnet: 26,
  xpla: 28,
  btc: 29,
  base: 30,
  sei: 32,
  wormchain: 3104,
  sepolia: 10002,
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
  | "arbitrum"
  | "optimism"
  | "gnosis"
  | "base"
  | "sepolia";

/**
 *
 * All the Solana-based chain names that Wormhole supports
 */
export type SolanaChainName = "solana" | "pythnet";

export type CosmWasmChainName =
  | "terra"
  | "terra2"
  | "injective"
  | "xpla"
  | "sei";
export type TerraChainName = "terra" | "terra2";

export type Contracts = {
  core: string | undefined;
  token_bridge: string | undefined;
  nft_bridge: string | undefined;
};

export type ChainContracts = {
  [chain in ChainName]: Contracts;
};

export type Network = "MAINNET" | "TESTNET" | "DEVNET";

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
    core: "842125965",
    token_bridge: "842126029",
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
    core: "contract.wormhole_crypto.near",
    token_bridge: "contract.portalbridge.near",
    nft_bridge: undefined,
  },
  injective: {
    core: "inj17p9rzwnnfxcjp32un9ug7yhhzgtkhvl9l2q74d",
    token_bridge: "inj1ghd753shjuwexxywmgs4xz7x2q732vcnxxynfn",
    nft_bridge: undefined,
  },
  osmosis: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  aptos: {
    core: "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
    token_bridge:
      "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f",
    nft_bridge:
      "0x1bdffae984043833ed7fe223f7af7a3f8902d04129b14f801823e64827da7130",
  },
  sui: {
    core: "0xaeab97f96cf9877fee2883315d459552b2b921edc16d7ceac6eab944dd88919c",
    token_bridge:
      "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9",
    nft_bridge: undefined,
  },
  moonbeam: {
    core: "0xC8e2b0cD52Cf01b0Ce87d389Daa3d414d4cE29f3",
    token_bridge: "0xb1731c586ca89a23809861c6103f0b96b3f57d92",
    nft_bridge: "0x453cfbe096c0f8d763e8c5f24b441097d577bde2",
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
  arbitrum: {
    core: "0xa5f208e072434bC67592E4C49C1B991BA79BCA46",
    token_bridge: "0x0b2402144Bb366A632D14B83F244D2e0e21bD39c",
    nft_bridge: "0x3dD14D553cFD986EAC8e3bddF629d82073e188c8",
  },
  optimism: {
    core: "0xEe91C335eab126dF5fDB3797EA9d6aD93aeC9722",
    token_bridge: "0x1D68124e65faFC907325e3EDbF8c4d84499DAa8b",
    nft_bridge: "0xfE8cD454b4A1CA468B57D79c0cc77Ef5B6f64585",
  },
  gnosis: {
    core: "0xa321448d90d4e5b0A732867c18eA198e75CAC48E",
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  pythnet: {
    core: "H3fxXJ86ADW2PNuDDmZJg6mzTtPxkYCpNuQUTgmJ7AjU",
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  xpla: {
    core: "xpla1jn8qmdda5m6f6fqu9qv46rt7ajhklg40ukpqchkejcvy8x7w26cqxamv3w",
    token_bridge:
      "xpla137w0wfch2dfmz7jl2ap8pcmswasj8kg06ay4dtjzw7tzkn77ufxqfw7acv",
    nft_bridge: undefined,
  },
  btc: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  base: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  sei: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  wormchain: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  sepolia: {
    // This is testnet only.
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
    core: "wormhole.wormhole.testnet",
    token_bridge: "token.wormhole.testnet",
    nft_bridge: undefined,
  },
  injective: {
    core: "inj1xx3aupmgv3ce537c0yce8zzd3sz567syuyedpg",
    token_bridge: "inj1q0e70vhrv063eah90mu97sazhywmeegp7myvnh",
    nft_bridge: undefined,
  },
  osmosis: {
    core: "osmo1hggkxr0hpw83f8vuft7ruvmmamsxmwk2hzz6nytdkzyup9krt0dq27sgyx",
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  aptos: {
    core: "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
    token_bridge:
      "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f",
    nft_bridge: undefined,
  },
  sui: {
    core: "0x69ae41bdef4770895eb4e7aaefee5e4673acc08f6917b4856cf55549c4573ca8",
    token_bridge:
      "0x32422cb2f929b6a4e3f81b4791ea11ac2af896b310f3d9442aa1fe924ce0bab4",
    nft_bridge: undefined,
  },
  moonbeam: {
    core: "0xa5B7D85a8f27dd7907dc8FdC21FA5657D5E2F901",
    token_bridge: "0xbc976D4b9D57E57c3cA52e1Fd136C45FF7955A96",
    nft_bridge: "0x98A0F4B96972b32Fcb3BD03cAeB66A44a6aB9Edb",
  },
  neon: {
    core: "0x268557122Ffd64c85750d630b716471118F323c8",
    token_bridge: "0xEe3dB83916Ccdc3593b734F7F2d16D630F39F1D0",
    nft_bridge: "0x66E5BcFD45D2F3f166c567ADa663f9d2ffb292B4",
  },
  terra2: {
    core: "terra19nv3xr5lrmmr7egvrk2kqgw4kcn43xrtd5g0mpgwwvhetusk4k7s66jyv0",
    token_bridge:
      "terra1c02vds4uhgtrmcw7ldlg75zumdqxr8hwf7npseuf2h58jzhpgjxsgmwkvk",
    nft_bridge: undefined,
  },
  arbitrum: {
    core: "0xC7A204bDBFe983FCD8d8E61D02b475D4073fF97e",
    token_bridge: "0x23908A62110e21C04F3A4e011d24F901F911744A",
    nft_bridge: "0xEe3dB83916Ccdc3593b734F7F2d16D630F39F1D0",
  },
  optimism: {
    core: "0x6b9C8671cdDC8dEab9c719bB87cBd3e782bA6a35",
    token_bridge: "0xC7A204bDBFe983FCD8d8E61D02b475D4073fF97e",
    nft_bridge: "0x23908A62110e21C04F3A4e011d24F901F911744A",
  },
  gnosis: {
    core: "0xE4eacc10990ba3308DdCC72d985f2a27D20c7d03",
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  pythnet: {
    core: "EUrRARh92Cdc54xrDn6qzaqjA77NRrCcfbr8kPwoTL4z",
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  xpla: {
    core: "xpla1upkjn4mthr0047kahvn0llqx4qpqfn75lnph4jpxfn8walmm8mqsanyy35",
    token_bridge:
      "xpla1kek6zgdaxcsu35nqfsyvs2t9vs87dqkkq6hjdgczacysjn67vt8sern93x",
    nft_bridge: undefined,
  },
  btc: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  base: {
    core: "0x23908A62110e21C04F3A4e011d24F901F911744A",
    token_bridge: "0xA31aa3FDb7aF7Db93d18DDA4e19F811342EDF780",
    nft_bridge: "0xF681d1cc5F25a3694E348e7975d7564Aa581db59",
  },
  sei: {
    core: "sei1nna9mzp274djrgzhzkac2gvm3j27l402s4xzr08chq57pjsupqnqaj0d5s",
    token_bridge:
      "sei1jv5xw094mclanxt5emammy875qelf3v62u4tl4lp5nhte3w3s9ts9w9az2",
    nft_bridge: undefined,
  },
  wormchain: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  sepolia: {
    core: "0x4a8bc80Ed5a4067f1CCf107057b8270E0cC11A78",
    token_bridge: "0xDB5492265f6038831E89f495670FF909aDe94bd9",
    nft_bridge: "0x6a0B52ac198e4870e5F3797d5B403838a5bbFD99",
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
    nft_bridge: "terra1plju286nnfj3z54wgcggd4enwaa9fgf5kgrgzl",
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
    core: "0xde0036a9600559e295d5f6802ef6f3f802f510366e0c23912b0655d972166017",
    token_bridge:
      "0x84a5f374d29fc77e370014dce4fd6a55b58ad608de8074b0be5571701724da31",
    nft_bridge:
      "0x46da3d4c569388af61f951bdd1153f4c875f90c2991f6b2d0a38e2161a40852c",
  },
  sui: {
    core: "0x5a5160ca3c2037f4b4051344096ef7a48ebf4400b3f385e57ea90e1628a8bde0", // wormhole module State object ID
    token_bridge:
      "0xa6a3da85bbe05da5bfd953708d56f1a3a023e7fb58e5a824a3d4de3791e8f690", // token_bridge module State object ID
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
    core: "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
    token_bridge:
      "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6",
    nft_bridge: undefined,
  },
  arbitrum: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  optimism: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  gnosis: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  pythnet: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  xpla: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  btc: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  base: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  sei: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  wormchain: {
    core: "wormhole1ap5vgur5zlgys8whugfegnn43emka567dtq0jl",
    token_bridge: "wormhole1zugu6cajc4z7ue29g9wnes9a5ep9cs7yu7rn3z",
    nft_bridge: undefined,
  },
  sepolia: {
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
export const CHAIN_ID_ARBITRUM = CHAINS["arbitrum"];
export const CHAIN_ID_OPTIMISM = CHAINS["optimism"];
export const CHAIN_ID_GNOSIS = CHAINS["gnosis"];
export const CHAIN_ID_PYTHNET = CHAINS["pythnet"];
export const CHAIN_ID_XPLA = CHAINS["xpla"];
export const CHAIN_ID_BTC = CHAINS["btc"];
export const CHAIN_ID_BASE = CHAINS["base"];
export const CHAIN_ID_SEI = CHAINS["sei"];
export const CHAIN_ID_WORMCHAIN = CHAINS["wormchain"];
export const CHAIN_ID_SEPOLIA = CHAINS["sepolia"];

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

export type CosmWasmChainId = typeof CHAINS[CosmWasmChainName];

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

export function toCosmWasmChainId(
  chainName: CosmWasmChainName
): CosmWasmChainId {
  return CHAINS[chainName];
}

export function coalesceCosmWasmChainId(
  chain: CosmWasmChainId | CosmWasmChainName
): CosmWasmChainId {
  // this is written in a way that for invalid inputs (coming from vanilla
  // javascript or someone doing type casting) it will always return undefined.
  return typeof chain === "number" && isCosmWasmChain(chain)
    ? chain
    : toCosmWasmChainId(chain);
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
    chainId === CHAIN_ID_ARBITRUM ||
    chainId === CHAIN_ID_OPTIMISM ||
    chainId === CHAIN_ID_GNOSIS ||
    chainId === CHAIN_ID_BASE ||
    chainId === CHAIN_ID_SEPOLIA
  ) {
    return isEVM(chainId);
  } else {
    return notEVM(chainId);
  }
}

export function isCosmWasmChain(
  chain: ChainId | ChainName
): chain is CosmWasmChainId | CosmWasmChainName {
  const chainId = coalesceChainId(chain);
  return (
    chainId === CHAIN_ID_TERRA ||
    chainId === CHAIN_ID_TERRA2 ||
    chainId === CHAIN_ID_INJECTIVE ||
    chainId === CHAIN_ID_XPLA ||
    chainId === CHAIN_ID_SEI
  );
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

export const APTOS_DEPLOYER_ADDRESS =
  "0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b";
export const APTOS_DEPLOYER_ADDRESS_DEVNET =
  "277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b";
export const APTOS_TOKEN_BRIDGE_EMITTER_ADDRESS =
  "0000000000000000000000000000000000000000000000000000000000000001";

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
