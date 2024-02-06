import { toUtf8 } from "@cosmjs/encoding";
import {
  getWallet,
  getWormchainSigningClient,
} from "@wormhole-foundation/wormchain-sdk";
import { ZERO_FEE } from "@wormhole-foundation/wormchain-sdk/lib/core/consts";
import "dotenv/config";

async function main() {
  /* Set up cosmos client & wallet */

  const wallet = await getWallet(process.env.MNEMONIC);
  const client = await getWormchainSigningClient(
    process.env.WORMCHAIN_HOST,
    wallet
  );

  // there are several Cosmos chains in devnet, so check the config is as expected
  let id = await client.getChainId();
  if (id !== "wormchain") {
    throw new Error(
      `Wormchain CosmWasmClient connection produced an unexpected chainID: ${id}`
    );
  }

  const signers = await wallet.getAccounts();
  const signer = signers[0].address;
  console.log("wormchain contract deployer is: ", signer);

  const msg = client.wasm.msgExecuteContract({
    sender: signer,
    contract:
      "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465",
    msg: toUtf8(
      JSON.stringify({
        submit_vaas: {
          vaas: [
            Buffer.from(
              // This is the mainnet governance VAA for registering the Sui Token Bridge signed by the Guardians https://api.wormscan.io/api/v1/vaas/1/0000000000000000000000000000000000000000000000000000000000000004/8526464727442833848
              "01000000030d00a42d36048a27a763413bfa1d261daeb86f7feed7a9850d9915d8c44acee97dd4327c0351648ac4d496050e6af6534badd2b86416a5675f373e5a5ebf876505700002349ec8ae9aeef147e87b2b06a475d6ec760c20680285ecdd6b7b400cbd67504232e0cae965d27dc9f0a83fbed40ec0a759bd7ad9a014a69c90364a1df88048dc000323d01a78c01887d981e22ffb0f91d02c1d7f015393513db3f215af64bd9cb74829eb08fc9895f8b1247933dd23c2b71ac1bf5bbefb8bea829ab2f6b94317eaa901040305b91b17227313395a861365c8b110414e961ccf25a2b645226e6a307e488f6dace0896c354425f5dfb6b0cb968ef1f752653cd85179b9344f215b555601dd01060dcfd80b54b6f43e502b9628a5a2b92b453fa96718397ecbe0e279495ff37c0835fff5ee7f3b749de287a0c3440105ddd705d06975d56384792373645b7787df000715985ff2cf28e3a8062d9ffc7ef69fbd7c56082f0938cc564586d0dbcacf14986579e7d7a8629dc8a9cfcbf0c97c46ae6492d05c5fba193400746a43b6f1123d000a5db8adfca6d43dd345e130fa0cae250ac7cfa364a29d47cf219ce2b50d6f930a0e1bc399b5b92cdec3d00fcf6f2c7f0732996344812dae85afcfa077c67d94b8010bf68fe7c2ed3aaa180b01ba28052fa63d72509e642bf45f8b5c14c582d8e6eb99514d41ab7fd3f6451470e02a054a3630e347020b6330a8ec23efdc3e4da4550b010d7f3ec58dbb8ae21a2fb71941ed80d646469f1992e7fdc32706c327bfbe01b98011e8b377fce487237f9238fe9af09991f5da11d85aba5a4a81e99df8d066aead010e18e7de979a55bd568b26754fdd7d9e7b03572d742e5657f944ab35b44398a40e07a0c2399e13a244277138375e7e980bf6b666f39bc2f86afd2605f0249a5a53000f77d089279a354b7faa1f3fdc084f6e0ef684d9bcce8d9fb11b5568c0d0b215f15d54cc4383e1b7112fadc238f750b885f5f81a21f84e00ec4487a8064386cc2e01106d9d3067e19413e985f76852eb0cdd071fef659540ddf3a9d5610d492a68a13c61cd109f64c977c1274f9782dcbddfa46ee94331e02f98ba8fb37e22300bd63e0111819a499e30feb82190736054d2993918aeb591e3098b4df77630e93512fec4c122f2cccfe88f2b735f42a06571944d800f3dfcb07de7956330515ddb3c9a41360000000000764b7752000100000000000000000000000000000000000000000000000000000000000000047654167e9520c1b820000000000000000000000000000000000000000000546f6b656e4272696467650100000015ccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5",
              "hex"
            ).toString("base64"),
          ],
        },
      })
    ),
    funds: [],
  });
  const res = await client.signAndBroadcast(signer, [msg], {
    ...ZERO_FEE,
    gas: "10000000",
  });
  console.log(res);
}

try {
  main();
} catch (e: any) {
  if (e?.message) {
    console.error(e.message);
  }
  throw e;
}
