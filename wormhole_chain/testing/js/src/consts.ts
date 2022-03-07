export const TEST_WALLET_MNEMONIC_1 =
  "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius";
export const TEST_WALLET_MNEMONIC_2 =
  "maple pudding enjoy pole real rabbit soft make square city wrestle area aisle dwarf spike voice over still post lend genius bitter exit shoot";

export const TEST_WALLET_ADDRESS_1 =
  "wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq";

export const DEVNET_GUARDIAN_PUBLIC_KEY =
  "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe";
export const DEVNET_GUARDIAN_PRIVATE_KEY =
  "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";
//This assume guardian 1 uses test wallet 1.
export const GUARDIAN_VALIDATOR_PUBLIC_KEY =
  "wormholevaloper1cyyzpxplxdzkeea7kwsydadg87357qna87hzv8";

export const DEVNET_GUARDIAN2_PUBLIC_KEY = "";
export const DEVNET_GUARDIAN2_PRIVATE_KEY = "";
//This assume guardian 1 uses test wallet 1.
export const GUARDIAN_VALIDATOR2_PUBLIC_KEY = "";

//This is a VAA in hex which is for guardian set 2, where Guardian 2 is the only active guardian.
export const GUARDIAN2_UPGRADE_VAA = "";

export const NODE_URL = "http://localhost:1317"; // TODO kube support
export const TENDERMINT_URL = "http://localhost:26657";
export const FAUCET_URL = "http://localhost:4500";

export const HOLE_DENOM = "uhole";
export const ADDRESS_PREFIX = "wormhole";
export const OPERATOR_PREFIX = "wormholevaloper";

export const DEVNET_SOLT = "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ";

//This is a transfer for 100 SOLT to Chain ID 2,
//And recipient address wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq, which should be TEST_WALLET_1
// Will need to update this with the real chain ID
export const TEST_TRANSFER_VAA =
  "010000000001007d204ad9447c4dfd6be62406e7f5a05eec96300da4048e70ff530cfb52aec44807e98194990710ff166eb1b2eac942d38bc1cd6018f93662a6578d985e87c8d0016221346b0000b8bd0001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f0000000000000003200100000000000000000000000000000000000000000000000000000002540be400165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3010001000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d00020000000000000000000000000000000000000000000000000000000000000000";
