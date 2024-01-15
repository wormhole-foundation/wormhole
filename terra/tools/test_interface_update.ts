// Run with MNEMONIC="" npx tsx test_interface_update.ts

// The intention of this script is to test the CW < 1 migration to CW > 1.
// 1. Instantiate new core and token bridge contracts from the existing Code IDs, using the devnet Guardian key.
// 2. Migrate those contracts to the same code IDs, triggering any migration-specific effect they may have had.
// 3. Register a foreign bridge and asset.
// 4. Send a foreign asset in and out. (actually, can't create a foreign asset so nvm)
// 5. Store the updated CW > 1 code to new code IDs.
// 6. Upgrade the contracts to the new code IDs.
// 7. Upgrade the contracts to the new code IDs again.
// 8. Send the foreign asset from step 4 in and out again.
// 9. Register another foreign bridge and asset.
// 10. Send a new foreign asset in and out. (skipping since 8 was new anyway due to being unable to complete 4)
// 11. Attest a native token.
// 12. Deposit and withdraw a native token. (This is broken in mainnet, but should be fixed with this upgrade.)
// 13. Send a native token out and back. (This is broken in mainnet, but should be fixed with this upgrade.)
// 14. Confirm that a VAA redeemed before the upgrade can't be redeemed again (like from step 3)
// 15. Send a 20-byte addressed native CW20 out and in
// 16. Send a 32-byte addressed native CW20 out and in

import "dotenv/config";
import {
  Fee,
  LCDClient,
  MnemonicKey,
  MsgUpdateContractAdmin,
} from "@terra-money/terra.js";
import {
  MsgInstantiateContract,
  MsgExecuteContract,
  MsgStoreCode,
} from "@terra-money/terra.js";
import { readFileSync } from "fs";
import { Bech32, toHex } from "@cosmjs/encoding";
import { zeroPad } from "ethers/lib/utils.js";

// gas estimation wasn't working, so you'll find many hardcoded values in here
// YMMV

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// look, broadcast and broadcastBlock still resulted in sequence mismatches
// and nobody has time for that
async function broadcastAndWait(terra, tx) {
  const response = await terra.tx.broadcast(tx);
  if (response?.code !== 0) {
    console.error(response);
    throw new Error(
      `Transaction failed https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
    );
  }
  let currentHeight = (await terra.tendermint.blockInfo()).block.header.height;
  while (currentHeight <= response.height) {
    await sleep(100);
    currentHeight = (await terra.tendermint.blockInfo()).block.header.height;
  }
  return response;
}

// Terra addresses are "human-readable", but for cross-chain registrations, we
// want the "canonical" version
function convert_terra_address_to_hex(human_addr) {
  return "0x" + toHex(zeroPad(Bech32.decode(human_addr).data, 32));
}

async function submitCoreBridgeVAA(vaa: string) {
  const tx = await wallet.createAndSignTx({
    msgs: [
      new MsgExecuteContract(wallet.key.accAddress, addressCoreBridge, {
        submit_v_a_a: {
          vaa: Buffer.from(vaa, "hex").toString("base64"),
        },
      }),
    ],
    memo: "",
    fee: new Fee(1000000, { uluna: 50_000_000 }),
  });
  const response = await broadcastAndWait(terra, tx);
  return response?.txhash;
}

async function submitTokenBridgeVAA(vaa: string) {
  const tx = await wallet.createAndSignTx({
    msgs: [
      new MsgExecuteContract(wallet.key.accAddress, addressTokenBridge, {
        submit_vaa: {
          data: Buffer.from(vaa, "hex").toString("base64"),
        },
      }),
    ],
    memo: "",
    fee: new Fee(1000000, { uluna: 50_000_000 }),
  });
  const response = await broadcastAndWait(terra, tx);
  return response?.txhash;
}

/* Set up terra client & wallet */

const terra = new LCDClient({
  URL: "https://terra-classic-lcd.publicnode.com",
  chainID: "columbus-5",
  isClassic: false,
});

if (!process.env.MNEMONIC) {
  throw new Error("MNEMONIC is required");
}
const wallet = terra.wallet(
  new MnemonicKey({
    mnemonic: process.env.MNEMONIC,
  })
);

const existingCodeIds = {
  "wormhole.wasm": 557, // current wasm
  "token_bridge_terra.wasm": 6097, // current wasm
  "cw20_wrapped.wasm": 767, // current wasm
};

// default addresses from first run
let addressCoreBridge: string =
  "terra1xd3f9g77qd5774kkepnn7wndjdlqujsvp5kg0pj7yp55crgvju7snjjgxc";
let addressTokenBridge: string =
  "terra1kxp07aarhyurar4r4ertszlvhhjmt07j3fusdfu9pj4akkrdcdys9694q2";

async function deployCode(file: string) {
  const contract_bytes = readFileSync(`../artifacts/${file}`);
  console.log(`Storing WASM: ${file} (${contract_bytes.length} bytes)`);

  const store_code = new MsgStoreCode(
    wallet.key.accAddress,
    contract_bytes.toString("base64")
  );

  const tx = await wallet.createAndSignTx({
    msgs: [store_code],
    memo: "",
    fee: new Fee(5000000, { uluna: 200_000_000 }),
  });

  const rs = await broadcastAndWait(terra, tx);
  console.log(rs.raw_log);
  console.log(
    `Deployed ${file} https://finder.terraclassic.community/mainnet/tx/${rs?.txhash}`
  );
  const ci = /"code_id","value":"([^"]+)/gm.exec(rs.raw_log)?.[1];
  if (!ci) {
    throw new Error("Could not parse code ID from raw_log");
  }
  return parseInt(ci);
}

async function instantiate(contract, inst_msg, label) {
  var address;
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          wallet.key.accAddress,
          wallet.key.accAddress,
          existingCodeIds[contract],
          inst_msg,
          undefined,
          label
        ),
      ],
      memo: "",
      fee: new Fee(5000000, { uluna: 200_000_000 }),
    })
    .then((tx) => broadcastAndWait(terra, tx))
    .then((rs) => {
      address = /"_contract_address","value":"([^"]+)/gm.exec(rs.raw_log)?.[1];
    });
  console.log(
    `Instantiated ${contract} at ${address} (${convert_terra_address_to_hex(
      address
    )})`
  );
  return address;
}

async function step1() {
  const govChain = 1;
  const govAddress =
    "0000000000000000000000000000000000000000000000000000000000000004";

  // devnet guardian public key
  const init_guardians = ["beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"];

  addressCoreBridge = await instantiate(
    "wormhole.wasm",
    {
      gov_chain: govChain,
      gov_address: Buffer.from(govAddress, "hex").toString("base64"),
      guardian_set_expirity: 86400,
      initial_guardian_set: {
        addresses: init_guardians.map((hex) => {
          return {
            bytes: Buffer.from(hex, "hex").toString("base64"),
          };
        }),
        expiration_time: 0,
      },
    },
    "wormholeTest"
  );

  addressTokenBridge = await instantiate(
    "token_bridge_terra.wasm",
    {
      gov_chain: govChain,
      gov_address: Buffer.from(govAddress, "hex").toString("base64"),
      wormhole_contract: addressCoreBridge,
      wrapped_asset_code_id: existingCodeIds["cw20_wrapped.wasm"],
    },
    "tokenBridgeTest"
  );
}

async function step2() {
  async function updateAdmin(contract: string) {
    const tx = await wallet.createAndSignTx({
      msgs: [
        new MsgUpdateContractAdmin(wallet.key.accAddress, contract, contract),
      ],
      memo: "",
      fee: new Fee(200000, { uluna: 10_000_000 }),
    });
    const response = await broadcastAndWait(terra, tx);
    console.log(
      `Updated ${contract} admin to itself https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
    );
  }
  await updateAdmin(addressCoreBridge);
  await updateAdmin(addressTokenBridge);

  {
    // worm generate upgrade -c terra -a 557 -m Core -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const upgradeCoreVaa =
      "01000000000100873cedead0c5a60da23bc23b3705e3b5b0600630d61a955c308307565e54b0cc14914495fc0eb39aee667f81adde5f49aec8bb774bb34121130e24ad6ecacb4c000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000005300d3f0000000000000000000000000000000000000000000000000000000000436f7265010003000000000000000000000000000000000000000000000000000000000000022d";
    const txhash = await submitCoreBridgeVAA(upgradeCoreVaa);
    console.log(
      `Upgraded core bridge https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }

  {
    // worm generate upgrade -c terra -a 6097 -m TokenBridge -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const upgradeTokenBridgeVaa =
      "01000000000100c6267b6d66e0ff7fd1b625a94402a5c3ea04cbda7157ab42e172a5844f0cdc495e327c9e1cb9acf2f6c07597d8e746f593253cdaed451994765de1b8ae6fc3d9010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000001be411000000000000000000000000000000000000000000000546f6b656e42726964676502000300000000000000000000000000000000000000000000000000000000000017d1";
    // Unable to run this... 'failed to execute message; message index: 0: Generic error: Querier contract error: codespace: wasm, code: 9: execute wasm contract failed'
    // const txhash = await submitTokenBridgeVAA(upgradeCoreVaa);
    // console.log(
    //   `Upgraded token bridge https://finder.terraclassic.community/mainnet/tx/${txhash}`
    // );
  }
}

const registerAvalanche =
  "010000000001000f4c334aad9d3a3a9025654bdc5b6c544962683d3be6f616e2e8bdb4f0c2d292423c12b62ea1761c208fb2f3c3734dc25110d7ec7bed9d5db64f2f4babd17eaf010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002cb3d3900000000000000000000000000000000000000000000546f6b656e42726964676501000000060000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052";
async function step3() {
  {
    // Avalanche mainnet token bridge registration
    // worm generate registration -c avalanche -a 0e082F06FF657D94310cB8cE8B0D9a04541d8052 -m TokenBridge -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const txhash = await submitTokenBridgeVAA(registerAvalanche);
    console.log(
      `Registered Avalanche https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }
  {
    // WAVAX attestation
    // worm edit-vaa -v AQAAAAMNAaDxJk8ByJTD1FsV4F2ZHyoV1/ZgDvhf+CsKmVTXt5PsdmTurfZKrcr752G94X6ghbT46rxr0bPy4NkoX7MKQJwAAk/crb5HuZnwmD+C6MFIeIBySqvxDQ/mssXHthZrQFsyLvmGZ8COYM9lZEb9pCoiBxuni4pG8gfSH/hEbIkLfX8AA43O2Ymg124dnkzw1E3dIhMt0qKMJozhXdzL/09cB4aJDWX5qxColViikLBqFqM/H/TUasl6Mq2Adhxb6U0h9LEABHAKcM3HK4mPKvPO/tLHW1obzvfp1Dr9QnrkXa+Nh9NMCzHI/JjaUxqUv+9CWntOX3UkGhSkcG0OnjWT79Z1B6kBBd1hBteD0zsrftlGvKzBC9uK3tJ6qwC/YOt0G5criYsqetYTgYnXpZiNrUBdKsqloNUpYHkkE/ieDDFUYxu8IwQAB6NfbF5YrI97b8EoQpz6IW93rJyymNFjYZpFQ3g4gDnoNQpqU0x6xH9NAVq1jJa1sdXjqFnDFDIPJxVHZvX0nAkBCl+kaIABlDgB2TyYvNop2DHKdLj1/NFVqspxg2OZ+PFpcYWZnhIJpNR2YTS7fppI9chxGGoVaoF2ufGAA155kiYAC/2UFf45V9DrsDiW5oJd74PXz3GvbB7G78me4NBYTlZ6E2U9RUQ8ib2W2G7NyQAh2rA/0v9oVbFMOjHaovp4E0kADIUXrdHj7UphaDMNVmFI8uPbLTNHh1j0yweg9TorA3i9QVG++wJ8mFW/+qCMhcw4M5MkJKhbohEQSTdiENir/hkBDgmH2Mb76kLUpy72LBbuQmbzTxb/72PRSqIpMLQzaCbmMGlpUaK3vgZETvUZjjBgt4pMDzki6KjZ67bXhp8FsFoAD/VLPF3juH6/5ulXqmMKP3VJts/ZTzhvn7yRV+YA5YjXUdzUh3pYbraURIsrXvJKzp9+bLzLh2OX/8XRU3bPfLgAEBhwvU9CUEiZ5+3jwSEl3u7rF7ADfUCFeSc/Tb46brgtZNf06ps56C9GKa/c6pSIXEKFjm6AyptXkCeIA9/MdioBEZcYyMrEYkW+gUFGCAaoSSyQmdTmydElc514U3inTM9FETZNTCFmxK/5VHnbEgLKBiDCXL6VDqr8pqBe5DzPxtoAZXIwrDh5AAAABgAAAAAAAAAAAAAAAA4ILwb/ZX2UMQy4zosNmgRUHYBSAAAAAAABqd0BAgAAAAAAAAAAAAAAALMfZqo8HnhTY/CHWht04nuF/WbHAAYSV0FWQVgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXcmFwcGVkIEFWQVgAAAAAAAAAAAAAAAAAAAAAAAAAAA== --gs cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const wavaxAttestation =
      "01000000000100b1cc54a6b6a3ee8baadc39f7f59b6074ae647d811ffb3deac1036231dfdc26f02a0e6fa6033f549efc358e7443d979bd261ab3ad8f74ce6f90c14cb136a6832701657230ac3879000000060000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052000000000001a9dd0102000000000000000000000000b31f66aa3c1e785363f0875a1b74e27b85fd66c700061257415641580000000000000000000000000000000000000000000000000000005772617070656420415641580000000000000000000000000000000000000000";
    // Unable to run this... 'failed to execute message; message index: 0: dispatch: submessages: label is required: invalid request'
    // const txhash = await submitTokenBridgeVAA(wavaxAttestation);
    // console.log(
    //   `Created WAVAX https://finder.terraclassic.community/mainnet/tx/${txhash}`
    // );
  }
}

async function step4() {
  // cannot complete step 4 since unable to create the wrapped asset on the existing implementation
}

// default code id from first run
let newCodeIdCoreBridge: number = 8336;
let newCodeIdTokenBridge: number = 8337;

async function step5() {
  newCodeIdCoreBridge = await deployCode("wormhole.wasm");
  newCodeIdTokenBridge = await deployCode("token_bridge_terra.wasm");
}

async function step6() {
  {
    // worm generate upgrade -c terra -a 8333 -m Core -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const upgradeCoreVaa =
      "01000000000100184d8fd19d0156cbba2c04e93d9f6a13af14388866a6acadde0db5a323deb0c23e40fe42ab52c285453b76fc535a64aa0f9d97f90dc44ca1079c50b7f5a10dc0000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000005dbae8f0000000000000000000000000000000000000000000000000000000000436f72650100030000000000000000000000000000000000000000000000000000000000002090";
    const txhash = await submitCoreBridgeVAA(upgradeCoreVaa);
    console.log(
      `Upgraded core bridge https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }

  {
    // worm generate upgrade -c terra -a 8334 -m TokenBridge -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const upgradeTokenBridgeVaa =
      "010000000001007e4d9ebea907f55fcf011b7419812707200adc237c5159364486f95e182b98622eab205b0be2af96eb9a5010ab8b2ea3103c5ac5080a7ab9ce42a2bbfa70462f0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000000913c800000000000000000000000000000000000000000000546f6b656e4272696467650200030000000000000000000000000000000000000000000000000000000000002091";
    const txhash = await submitTokenBridgeVAA(upgradeTokenBridgeVaa);
    console.log(
      `Upgraded token bridge https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }
}

async function step7() {
  {
    // worm generate upgrade -c terra -a 8333 -m Core -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const upgradeCoreVaa =
      "0100000000010076c3cea65bb6a1657a5d4736133fcc9dd1c7ff363715e4d4291461e98430f3a7238f12d26bf3b7715915628ed178726d161e594d9722440e01f797658e52351b000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000005b35c780000000000000000000000000000000000000000000000000000000000436f72650100030000000000000000000000000000000000000000000000000000000000002090";
    const txhash = await submitCoreBridgeVAA(upgradeCoreVaa);
    console.log(
      `Upgraded core bridge https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }

  {
    // worm generate upgrade -c terra -a 8334 -m TokenBridge -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const upgradeTokenBridgeVaa =
      "01000000000100ae7e1701c5b289ecee8e75140050d5166f8683769e0b6c04f9623331f6c6bfcc068c7fb7ad723c991411bc8eca5a405a8614fe11aa4a9f9861e3a0af06a5c859010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000005c295db00000000000000000000000000000000000000000000546f6b656e4272696467650200030000000000000000000000000000000000000000000000000000000000002091";
    const txhash = await submitTokenBridgeVAA(upgradeTokenBridgeVaa);
    console.log(
      `Upgraded token bridge https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }
}

async function step8a() {
  {
    // WAVAX attestation
    // worm edit-vaa -v AQAAAAMNAaDxJk8ByJTD1FsV4F2ZHyoV1/ZgDvhf+CsKmVTXt5PsdmTurfZKrcr752G94X6ghbT46rxr0bPy4NkoX7MKQJwAAk/crb5HuZnwmD+C6MFIeIBySqvxDQ/mssXHthZrQFsyLvmGZ8COYM9lZEb9pCoiBxuni4pG8gfSH/hEbIkLfX8AA43O2Ymg124dnkzw1E3dIhMt0qKMJozhXdzL/09cB4aJDWX5qxColViikLBqFqM/H/TUasl6Mq2Adhxb6U0h9LEABHAKcM3HK4mPKvPO/tLHW1obzvfp1Dr9QnrkXa+Nh9NMCzHI/JjaUxqUv+9CWntOX3UkGhSkcG0OnjWT79Z1B6kBBd1hBteD0zsrftlGvKzBC9uK3tJ6qwC/YOt0G5criYsqetYTgYnXpZiNrUBdKsqloNUpYHkkE/ieDDFUYxu8IwQAB6NfbF5YrI97b8EoQpz6IW93rJyymNFjYZpFQ3g4gDnoNQpqU0x6xH9NAVq1jJa1sdXjqFnDFDIPJxVHZvX0nAkBCl+kaIABlDgB2TyYvNop2DHKdLj1/NFVqspxg2OZ+PFpcYWZnhIJpNR2YTS7fppI9chxGGoVaoF2ufGAA155kiYAC/2UFf45V9DrsDiW5oJd74PXz3GvbB7G78me4NBYTlZ6E2U9RUQ8ib2W2G7NyQAh2rA/0v9oVbFMOjHaovp4E0kADIUXrdHj7UphaDMNVmFI8uPbLTNHh1j0yweg9TorA3i9QVG++wJ8mFW/+qCMhcw4M5MkJKhbohEQSTdiENir/hkBDgmH2Mb76kLUpy72LBbuQmbzTxb/72PRSqIpMLQzaCbmMGlpUaK3vgZETvUZjjBgt4pMDzki6KjZ67bXhp8FsFoAD/VLPF3juH6/5ulXqmMKP3VJts/ZTzhvn7yRV+YA5YjXUdzUh3pYbraURIsrXvJKzp9+bLzLh2OX/8XRU3bPfLgAEBhwvU9CUEiZ5+3jwSEl3u7rF7ADfUCFeSc/Tb46brgtZNf06ps56C9GKa/c6pSIXEKFjm6AyptXkCeIA9/MdioBEZcYyMrEYkW+gUFGCAaoSSyQmdTmydElc514U3inTM9FETZNTCFmxK/5VHnbEgLKBiDCXL6VDqr8pqBe5DzPxtoAZXIwrDh5AAAABgAAAAAAAAAAAAAAAA4ILwb/ZX2UMQy4zosNmgRUHYBSAAAAAAABqd0BAgAAAAAAAAAAAAAAALMfZqo8HnhTY/CHWht04nuF/WbHAAYSV0FWQVgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXcmFwcGVkIEFWQVgAAAAAAAAAAAAAAAAAAAAAAAAAAA== --gs cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const wavaxAttestation =
      "01000000000100b1cc54a6b6a3ee8baadc39f7f59b6074ae647d811ffb3deac1036231dfdc26f02a0e6fa6033f549efc358e7443d979bd261ab3ad8f74ce6f90c14cb136a6832701657230ac3879000000060000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052000000000001a9dd0102000000000000000000000000b31f66aa3c1e785363f0875a1b74e27b85fd66c700061257415641580000000000000000000000000000000000000000000000000000005772617070656420415641580000000000000000000000000000000000000000";
    const txhash = await submitTokenBridgeVAA(wavaxAttestation);
    console.log(
      `Created WAVAX https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }
}

const wavaxAddress =
  "terra1kw3u6nyle39qhj4s725rg2lczdtscqkhfcrg33jvup00cm6ae3uqw9ewvw";

async function step8b() {
  // WAVAX transfer in
  // worm edit-vaa -v AQAAAAMNAOoKGMdKp40UoEoAQ0qZXVPawkpV4yZ4qCN7XC9KYV2VLLDcitjwRZ0rWh2xzFZVAntuxTSsl7sXXs5bkdKDNVEBAithN161Hu4+xIo7ADSNMTehvD70vg3bjB7/5cDt+ieYWZNeyhPgGlA/31y2vS5xtRGUOgCHoHVyDGTJ7RqRCpgBA6Z/g3R64BdryeqYDbGytx1uSNT7rbqadNkVb0BZvaJFAE5ITG9GW2bOZIAicg8iOcZw7WYa8a6vmuubKfzYp78BBI8OTu8tW72PzIspTYjEwinm7LpfgQ51ERIkAtFug2tmGZTwGiNRXoZeo9m1XpLgR/U01y6VQrIqUJasamVn6XwABv6qxtZkBfgT0T3nu6HEwv88CYXT/FXAYu/7HOVII/OneQQxIT9OS2BTiuMIWVcuvZ2qP2sdGhtOcX1f4crPCMcAB0yP5J7sQu+D2LDUsYjdELtMtj4KeWtjCkCBR6skWe/8IuRndXFvllNo0GRsB6A3XFnXReadRp5jH4A6t5UzS1UBCG6rkvPMMt7oUdSZBt9Lytd554Q5E310keqAB8+SXFumUY1y17U6KSgfTKgnWZBrTINj1RUw1ew8hElFziUKLDMADZE1+TeCwS+7W/59kz2mxBOgYa63j4pwdsB33BRGRfw9aYSQ3mkszyH+88bIKgjWzhJvRjs622L20QnBIbs5EAABDtf8OfUwbyAeOKM5D6m9pkDMlmz0hNwLa1Iuk2sQjLJsLRtbFYXScfgErmSkIgGGoMRtMt7Hk9pkebaD92skGOkADy10mYk73Lm360c2DZsnTCYsv0rKQ+MMrMverp9d8K5uALqZ3huvhJmTKtpNG/Dpzjsolk2aADcPHuEJudy4ZFEBEA19NY1c97Qo+VMwx3E/B6sEFszT8L93zPYhjBkWsuY9HwSsogKKb84170DApcKiXuz2jhw4e/eStObBBn7gc0AAESW+7FjIHqqFL35z/T+Mpq8lrvaPwiZJnKaMipWzre4RIRrBx/UKhGaADniFOOCc1J2/KMwH6E5WXFbO9pexAkoBEpxYKgkPcldCBmwI96Rk6lul3Rflkj3FrJvVrzJumyBffW+NwSdvFP0QltmzVEN39K9prXttF5bTlxcxaxpFU58AZXI6igwiAAAABgAAAAAAAAAAAAAAAA4ILwb/ZX2UMQy4zosNmgRUHYBSAAAAAAABqe0BAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABkAAAAAAAAAAAAAAAAsx9mqjweeFNj8IdaG3Tie4X9ZscABgAAAAAAAAAAAAAAAD1aJY70jTtGjw8Tlz+RsqmlzFPYAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA== --gs cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
  const wavaxTransfer =
    "0100000000010044dd5206ac105e99c39b2a368e510978f1e18cf61bfbfac399244c9bbcf23b3e61bd5e6ca098855263745890bae05e6450a9d4ccdb395ecd01471e5867db738a0065723a8a0c22000000060000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052000000000001a9ed01010000000000000000000000000000000000000000000000000000000000000064000000000000000000000000b31f66aa3c1e785363f0875a1b74e27b85fd66c700060000000000000000000000003d5a258ef48d3b468f0f13973f91b2a9a5cc53d800030000000000000000000000000000000000000000000000000000000000000000";
  const txhash = await submitTokenBridgeVAA(wavaxTransfer);
  console.log(
    `Redeemed WAVAX https://finder.terraclassic.community/mainnet/tx/${txhash}`
  );
}

async function step8c() {
  // WAVAX transfer out
  const amount = "100";
  const tx = await wallet.createAndSignTx({
    msgs: [
      new MsgExecuteContract(
        wallet.key.accAddress,
        wavaxAddress,
        {
          increase_allowance: {
            spender: addressTokenBridge,
            amount,
            expires: {
              never: {},
            },
          },
        },
        {}
      ),
      new MsgExecuteContract(
        wallet.key.accAddress,
        addressTokenBridge,
        {
          initiate_transfer: {
            asset: {
              amount,
              info: {
                token: {
                  contract_addr: wavaxAddress,
                },
              },
            },
            recipient_chain: 4,
            recipient: Buffer.from(
              "0000000000000000000000000000000000000000000000000000000000000000",
              "hex"
            ).toString("base64"),
            fee: "0",
            nonce: 0,
          },
        },
        {}
      ),
    ],
    memo: "",
    fee: new Fee(1000000, { uluna: 50_000_000 }),
  });
  const response = await broadcastAndWait(terra, tx);
  console.log(
    `Transferred WAVAX https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
  );
}

async function step9() {
  {
    // Fantom mainnet token bridge registration
    // worm generate registration -c fantom -a 7C9Fc5741288cDFdD83CeB07f3ea7e22618D79D2 -m TokenBridge -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const registerAvalanche =
      "01000000000100f4644a32805b1e24eb40d91ffa05196bfa2783c74448070661701db4168a3e55787e1bd1bc4b300309b85990a4a75c8f7ff420c90b5d5a853076e413a02ef98c000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004eadf0f00000000000000000000000000000000000000000000546f6b656e427269646765010000000a0000000000000000000000007c9fc5741288cdfdd83ceb07f3ea7e22618d79d2";
    const txhash = await submitTokenBridgeVAA(registerAvalanche);
    console.log(
      `Registered Fantom https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }
  {
    // WFTM attestation
    // worm edit-vaa -v AQAAAAMNAGZLTJJ7k9izhrCiPU4vVY7CYp2c5592bObi2AKLGlj4CCxEGiQMva1F92+4kiX96CQoHG6iGSLX/uIwWZ0LdxkBAkq/rbbsqsk/Ax1gcowejzzJXgvfuSReU/Ob2sj4om2UY/L5R9b15jNb9uFexIkXitEaMRAzy0pnX1dBHx8ATfkAAyLgNbwTYAuA+sY+99cvm+e7r4V98IgXC4abG18t2K1nR9wxLpUMzNVPqDXVO/nDzvdQGqmHuhsC6IcwBT4CrCIABGWCArnx4DzWkS1Nsr88S7OvVOAWvn7AxTlVo0HT7SSwYhb7TUP2vBwJbA6mC0lK0UeBM3P4yoAKiFwSgqYPaZwABmewrAPd+r8uBGivWWfgsV5MbqwDEPy34mf+4LyhKtlifQc/QA3Mm+ttykD4SBp4ylAbvuwLBu8lqGrlzb/MoIYBBxLvs7gB2//Qx0qbsf7Ujau3g3Ipl1G2KTid1Xbn+5JDXbeJYEZQ3r2oiMB9N7HUNlldVPf3Nxn3oaZRHUvHYJkACzsVuZ0g4d2I2/XlLoanV90ElTFPjfmfe6Yp2Sol4mqOcz1Y+7QNmMm7iaKX9V7zRxvr/h6TWLip/P+mHjHRoywADGY6Vws8S5crT4U38OQoMEt9zFrBi1fttzdzv5P4BBKkWqUGP5ma2QyM39i6sussCgeI5qJae4ro3xXos5Qs218BDfgNXK8IojRwZoL8H+JKVOBv2qVN8rsbIwAN42TM3olCAliJYhhou8iZyjdeztmQRQl+7YESz8DVgLRNYhfPRDQADlQw39Txk19urWtjwb5LgVIWKea8AQqyocY2f3hJqbt9HFzWhR6Ho+gJ/wr7OnVoe9IIn4W5mvyiJYOLrmmNhzgBEEIZ8n6Qevl1+dZR2bdZHn6Cqdk6/UiPcyWmTHrMC8YKD+tKj3eRPHDDldnEJ23tsrtLhiCZm2OhiONefWCv9q8BEQCzPrgDqun4qbYMwAP8Bw8e22fJ5AnCOj6EksJEeYO7Kgwiijbc12wHosCaFD5YZXTqGPJ9zot1XH1tcFzYePABEtMHLrXStHqUtHUaR2KtRTKOL71Yy89SY8lEe70JJw9lVMJggnBVTPAdjic/frvLXFOZWgHmVYxz3fCIJWb0EboAZXMyeXI3AAAACgAAAAAAAAAAAAAAAHyfxXQSiM392DzrB/PqfiJhjXnSAAAAAAAAe34BAgAAAAAAAAAAAAAAACG+Nw1TEvRMtCzjd7ybigzvGkyDAAoSV0ZUTQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXcmFwcGVkIEZhbnRvbQAAAAAAAAAAAAAAAAAAAAAAAA== --gs cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    const wftmAttestation =
      "01000000000100bc136a00a448e2cc01ae1b81ca9d37a301a98fa906893b0735817b79d614b8bb3ede50f3d125bf581bff3640abc4b9ca83171c5e7ff7963f4a9a5517b6cb0009006573327972370000000a0000000000000000000000007c9fc5741288cdfdd83ceb07f3ea7e22618d79d20000000000007b7e010200000000000000000000000021be370d5312f44cb42ce377bc9b8a0cef1a4c83000a125746544d00000000000000000000000000000000000000000000000000000000577261707065642046616e746f6d000000000000000000000000000000000000";
    const txhash = await submitTokenBridgeVAA(wftmAttestation);
    console.log(
      `Created WFTM https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }
}

async function step10() {
  // skipping step10 as already covered by 8
}

async function step11() {
  const tx = await wallet.createAndSignTx({
    msgs: [
      new MsgExecuteContract(
        wallet.key.accAddress,
        addressTokenBridge,
        {
          create_asset_meta: {
            asset_info: {
              native_token: { denom: "uluna" },
            },
            nonce: 0,
          },
        },
        {}
      ),
    ],
    memo: "",
    fee: new Fee(1000000, { uluna: 50_000_000 }),
  });
  const response = await broadcastAndWait(terra, tx);
  console.log(
    `Attested uluna https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
  );
}

async function step12() {
  {
    const tx = await wallet.createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          wallet.key.accAddress,
          addressTokenBridge,
          {
            deposit_tokens: {},
          },
          { uluna: 10000 }
        ),
      ],
      memo: "",
      fee: new Fee(200000, { uluna: 10_000_000 }),
    });
    const response = await broadcastAndWait(terra, tx);
    console.log(
      `Deposted uluna https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
    );
  }
  {
    const tx = await wallet.createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          wallet.key.accAddress,
          addressTokenBridge,
          {
            withdraw_tokens: {
              asset: {
                native_token: {
                  denom: "uluna",
                },
              },
            },
          },
          {}
        ),
      ],
      memo: "",
      fee: new Fee(200000, { uluna: 10_000_000 }),
    });
    const response = await broadcastAndWait(terra, tx);
    console.log(
      `Withdrew uluna https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
    );
  }
}

async function step13() {
  {
    const tx = await wallet.createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          wallet.key.accAddress,
          addressTokenBridge,
          {
            deposit_tokens: {},
          },
          { uluna: 10000 }
        ),
        new MsgExecuteContract(
          wallet.key.accAddress,
          addressTokenBridge,
          {
            initiate_transfer: {
              asset: {
                amount: "10000",
                info: {
                  native_token: {
                    denom: "uluna",
                  },
                },
              },
              recipient_chain: 2,
              recipient: Buffer.from(
                "0000000000000000000000000000000000000000000000000000000000000000",
                "hex"
              ).toString("base64"),
              fee: "0",
              nonce: 1,
            },
          },
          {} // no fee?
          // { uluna: 10050 } // fee + tax
        ),
      ],
      memo: "",
      fee: new Fee(500000, { uluna: 20_000_000 }),
    });
    const response = await broadcastAndWait(terra, tx);
    console.log(
      `Sent uluna https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
    );
  }

  // 10000 uluna from avax to terra classic
  {
    const transferVAA =
      "010000000001002e1a7f786c42d41047bc4a6c55c2210a00dbeae45d43ead884eb42c163a7d72708a089c1e93779adbcf0603fac904a77c47e0c49cd0663ab250c434f6ceeb1f600652719abea95000000060000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d80520000000000018b9401010000000000000000000000000000000000000000000000000000000000002710010000000000000000000000000000000000000000000000000000756c756e6100030000000000000000000000003d5a258ef48d3b468f0f13973f91b2a9a5cc53d800030000000000000000000000000000000000000000000000000000000000000000";
    const txhash = await submitTokenBridgeVAA(transferVAA);
    console.log(
      `Redeemed uluna https://finder.terraclassic.community/mainnet/tx/${txhash}`
    );
  }
}

async function step14() {
  try {
    await submitTokenBridgeVAA(registerAvalanche);
  } catch (e) {}
}

const cw20with20ByteAddress = "terra1hj8de24c3yqvcsv9r8chr03fzwsak3hgd8gv3m";

async function step15a() {
  const tx = await wallet.createAndSignTx({
    msgs: [
      new MsgExecuteContract(
        wallet.key.accAddress,
        addressTokenBridge,
        {
          create_asset_meta: {
            asset_info: {
              token: {
                contract_addr: cw20with20ByteAddress,
              },
            },
            nonce: 0,
          },
        },
        {}
      ),
    ],
    memo: "",
    fee: new Fee(1000000, { uluna: 50_000_000 }),
  });
  const response = await broadcastAndWait(terra, tx);
  console.log(
    `Attested ${cw20with20ByteAddress} https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
  );
}

async function step15b() {
  const amount = "100";
  const tx = await wallet.createAndSignTx({
    msgs: [
      new MsgExecuteContract(
        wallet.key.accAddress,
        cw20with20ByteAddress,
        {
          increase_allowance: {
            spender: addressTokenBridge,
            amount,
            expires: {
              never: {},
            },
          },
        },
        {}
      ),
      new MsgExecuteContract(
        wallet.key.accAddress,
        addressTokenBridge,
        {
          initiate_transfer: {
            asset: {
              amount,
              info: {
                token: {
                  contract_addr: cw20with20ByteAddress,
                },
              },
            },
            recipient_chain: 4,
            recipient: Buffer.from(
              "0000000000000000000000000000000000000000000000000000000000000000",
              "hex"
            ).toString("base64"),
            fee: "0",
            nonce: 0,
          },
        },
        {}
      ),
    ],
    memo: "",
    fee: new Fee(1000000, { uluna: 50_000_000 }),
  });
  const response = await broadcastAndWait(terra, tx);
  console.log(
    `Transferred ${cw20with20ByteAddress} https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
  );
}

async function step15c() {
  const returnTransfer =
    "010000000001009b0cf0b08f933c518246235e5b43d33d20de3ecdd7cfb364ee4dd5b29dc14fb216aefdbd0d948eec506eb18ff348675a353c07e3ee244f224b2461b6c7983c650065723a8a0c22000000060000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052000000000001a9ed01010000000000000000000000000000000000000000000000000000000000000064000000000000000000000000bc8edcaab88900cc418519f171be2913a1db46e800030000000000000000000000003d5a258ef48d3b468f0f13973f91b2a9a5cc53d800030000000000000000000000000000000000000000000000000000000000000000";
  const txhash = await submitTokenBridgeVAA(returnTransfer);
  console.log(
    `Redeemed ${cw20with20ByteAddress} https://finder.terraclassic.community/mainnet/tx/${txhash}`
  );
}

const cw20with32ByteAddress =
  "terra1uac8wsrpm4xtwn7qx3rwz602ztsc4qcd9m8rkhx24ywqsxkpvlfq5ywat8";

async function step16prime() {
  const fileName = "cw20_base.wasm";
  const newTokenCodeId = await deployCode(fileName);
  console.log(`New test token code ID: ${newTokenCodeId}`);
  existingCodeIds[fileName] = newTokenCodeId;
  const addressTestToken = await instantiate(
    fileName,
    {
      name: "TEST",
      symbol: "TST",
      decimals: 6,
      initial_balances: [
        {
          address: wallet.key.accAddress,
          amount: "100000000",
        },
      ],
      mint: null,
    },
    "testToken"
  );
  console.log(
    `New test token instantiated https://finder.terraclassic.community/mainnet/address/${addressTestToken}`
  );
}

async function step16a() {
  const tx = await wallet.createAndSignTx({
    msgs: [
      new MsgExecuteContract(
        wallet.key.accAddress,
        addressTokenBridge,
        {
          create_asset_meta: {
            asset_info: {
              token: {
                contract_addr: cw20with32ByteAddress,
              },
            },
            nonce: 0,
          },
        },
        {}
      ),
    ],
    memo: "",
    fee: new Fee(1000000, { uluna: 50_000_000 }),
  });
  const response = await broadcastAndWait(terra, tx);
  console.log(
    `Attested ${cw20with32ByteAddress} https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
  );
}

async function step16b() {
  const amount = "100";
  const tx = await wallet.createAndSignTx({
    msgs: [
      new MsgExecuteContract(
        wallet.key.accAddress,
        cw20with32ByteAddress,
        {
          increase_allowance: {
            spender: addressTokenBridge,
            amount,
            expires: {
              never: {},
            },
          },
        },
        {}
      ),
      new MsgExecuteContract(
        wallet.key.accAddress,
        addressTokenBridge,
        {
          initiate_transfer: {
            asset: {
              amount,
              info: {
                token: {
                  contract_addr: cw20with32ByteAddress,
                },
              },
            },
            recipient_chain: 4,
            recipient: Buffer.from(
              "0000000000000000000000000000000000000000000000000000000000000000",
              "hex"
            ).toString("base64"),
            fee: "0",
            nonce: 0,
          },
        },
        {}
      ),
    ],
    memo: "",
    fee: new Fee(1000000, { uluna: 50_000_000 }),
  });
  const response = await broadcastAndWait(terra, tx);
  console.log(
    `Transferred ${cw20with32ByteAddress} https://finder.terraclassic.community/mainnet/tx/${response?.txhash}`
  );
}

async function step16c() {
  const returnTransfer =
    "010000000001004571e9f5012e31e7674d7e86adbcea5f3cfdbdef8ad1c95f46377144db94a4e77ecb390b4f1c1b01825b09bfbf89bb5597b45731d14c916db7b1817bbe0820000165723a8a0c22000000060000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052000000000001a9ed01010000000000000000000000000000000000000000000000000000000000000064e770774061dd4cb74fc03446e169ea12e18a830d2ece3b5ccaa91c081ac167d200030000000000000000000000003d5a258ef48d3b468f0f13973f91b2a9a5cc53d800030000000000000000000000000000000000000000000000000000000000000000";
  const txhash = await submitTokenBridgeVAA(returnTransfer);
  console.log(
    `Redeemed ${cw20with32ByteAddress} https://finder.terraclassic.community/mainnet/tx/${txhash}`
  );
}

async function main() {
  // await step1();
  console.log(
    `Core bridge:  https://finder.terraclassic.community/mainnet/address/${addressCoreBridge}`
  );
  console.log(
    `Token bridge: https://finder.terraclassic.community/mainnet/address/${addressTokenBridge}`
  );
  // await step2();
  // await step3();
  // await step4();
  // await step5();
  // console.log(`New core bridge code ID:  ${newCodeIdCoreBridge}`);
  // console.log(`New token bridge code ID: ${newCodeIdTokenBridge}`);
  // STOP HERE AND EDIT STEP 6 (also update the addresses and code IDs)
  // await step6();
  // await step7();
  // await step8a();
  // STOP HERE AND GATHER WAVAX ADDRESS
  // await step8b();
  // await step8c();
  // await step9();
  // await step10();
  // await step11();
  // await step12();
  // await step13();
  // await step14();
  // await step15a();
  // await step15b();
  // await step15c();
  // await step16prime();
  // await step16a();
  // await step16b();
  // await step16c();
}

main();
