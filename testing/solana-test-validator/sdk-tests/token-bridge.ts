import { expect } from "chai";
import { readFileSync } from "fs";
import * as web3 from "@solana/web3.js";
import {
  MockGuardians,
  MockEthereumEmitter,
  TokenBridgeGovernanceEmitter,
} from "../../../sdk/js/src/utils/mock";
import { parseVaa } from "../../../sdk/js/src/vaa/wormhole";

import {
  CORE_BRIDGE_ADDRESS,
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  GUARDIAN_SET_INDEX,
  LOCALHOST,
} from "./helpers/consts";
import {
  getPostedVaa,
  getGuardianSet,
  createSetFeesInstruction,
  createTransferFeesInstruction,
  createUpgradeGuardianSetInstruction,
  getBridgeInfo,
  feeCollectorKey,
  createBridgeFeeTransferInstruction,
} from "../../../sdk/js/src/solana/wormhole";
import { postVaa } from "../../../sdk/js/src/solana/sendAndConfirmPostVaa";
import { NodeWallet } from "../../../sdk/js/src/solana/utils";

describe("Wormhole (Core Bridge)", () => {
  const connection = new web3.Connection(LOCALHOST);

  const wallet = new NodeWallet(web3.Keypair.generate());

  // for signing wormhole messages
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  // for generating governance wormhole messages
  const governance = new TokenBridgeGovernanceEmitter(
    GOVERNANCE_EMITTER_ADDRESS
  );

  // hijacking the ethereum token bridge address for our fake emitter
  const ethereumTokenBridge = new MockEthereumEmitter(
    ETHEREUM_TOKEN_BRIDGE_ADDRESS
  );

  before("Airdrop SOL", async () => {
    await connection
      .requestAirdrop(wallet.key(), 1000 * web3.LAMPORTS_PER_SOL)
      .then(async (signature) => {
        await connection.confirmTransaction(signature);
        return signature;
      });
  });

  describe("Instruction 1: Post Message", () => {
    // TODO: add mock implementation contract and test that it can use the post_message instruction
  });
});
