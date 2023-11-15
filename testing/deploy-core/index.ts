// The following is a useful standalone script for deploying the core bridge somewhere, like your vanilla anvil setup
import { ethers_contracts } from "@certusone/wormhole-sdk";
import { Wallet, providers } from "ethers";

const rpc = process.env.RPC || "http://127.0.0.1:8545";
const pk = process.env.PRIVATE_KEY || "";
const mnemonic =
  process.env.MNEMONIC ||
  "test test test test test test test test test test test junk";

const initialSigners = process.env.INIT_SIGNERS
  ? JSON.parse(process.env.INIT_SIGNERS)
  : ["0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"];
const chainId = process.env.INIT_CHAIN_ID || "0x2";
const governanceChainId = process.env.INIT_GOV_CHAIN_ID || "0x1";
const governanceContract =
  process.env.INIT_GOV_CONTRACT ||
  "0x0000000000000000000000000000000000000000000000000000000000000004";

(async () => {
  const provider = new providers.JsonRpcProvider(rpc);
  const network = await provider.getNetwork();
  console.log("Connected to:", network);
  const evmChainId = `0x${network.chainId.toString(16)}`;
  const signer = (pk ? new Wallet(pk) : Wallet.fromMnemonic(mnemonic)).connect(
    provider
  );
  console.log("Using wallet:", await signer.getAddress());

  const setup = await new ethers_contracts.Setup__factory(signer).deploy();
  console.log("Setup:", setup.address);
  const implementation = await new ethers_contracts.Implementation__factory(
    signer
  ).deploy();
  console.log("Implementation:", implementation.address);

  const initData =
    ethers_contracts.Setup__factory.createInterface().encodeFunctionData(
      "setup",
      [
        implementation.address,
        initialSigners,
        chainId,
        governanceChainId,
        governanceContract,
        evmChainId,
      ]
    );
  const wormhole = await new ethers_contracts.Wormhole__factory(signer).deploy(
    setup.address,
    initData
  );
  console.log("Wormhole:", wormhole.address);
})();
