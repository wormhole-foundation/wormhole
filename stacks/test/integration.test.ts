import {
  broadcastTransaction,
  Cl,
  fetchCallReadOnlyFunction,
  fetchContractMapEntry,
  fetchNonce,
  makeContractCall,
  makeContractDeploy,
  Pc,
  privateKeyToAddress,
  TupleCV,
  UIntCV,
} from "@stacks/transactions";
import fs from "fs";
import path from "path";
import { describe, expect, it } from "vitest";
import { STACKS_API_URL, STACKS_PRIVATE_KEY } from "./lib/constants";
import {
  expectNoStacksVAA,
  expectVAA,
  waitForTransactionSuccess,
  wormhole,
} from "./lib/helpers";

const root = path.resolve(process.cwd(), "../");

describe("Stacks Wormhole Integration Tests", () => {
  it("should deploy stacks contracts", async () => {
    const ADDRESS = privateKeyToAddress(STACKS_PRIVATE_KEY, "devnet");

    const contractPath = path.resolve(root, "contracts");
    const dependencyPath = path.resolve(root, "contracts/dependencies");

    const rewriteClarity = (code: string) => {
      return code
        .replaceAll("SP2J933XB2CP2JQ1A4FGN8JA968BBG3NK3EKZ7Q9F", ADDRESS)
        .replaceAll("SP1E0XBN9T4B10E9QMR7XMFJPMA19D77WY3KP2QKC", ADDRESS)
        .replaceAll("SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM", ADDRESS);
    };

    const dependencyFiles = [
      "trait-sip-010.clar",
      "proposal-trait.clar",
      "extension-trait.clar",
      "executor-dao.clar",
      "trait-semi-fungible.clar",
      "token-amm-pool-v2-01.clar",
      "liquidity-locker.clar",
      "clarity-stacks.clar",
      "trait-flash-loan-user.clar",
      "amm-vault-v2-01.clar",
      "amm-registry-v2-01.clar",
      "amm-pool-v2-01.clar",
      "code-body-prover.clar",
      "clarity-stacks-helper.clar",
      "self-listing-helper-v3.clar",
      "hk-ecc-v1.clar",
      "hk-cursor-v2.clar",
      "hk-merkle-tree-keccak160-v1.clar",
    ].map((file) => path.join(dependencyPath, file));

    const contractFiles = [
      "wormhole-core-state.clar",
      "wormhole-trait-core-v2.clar",
      "wormhole-core-proxy-v2.clar",
      "wormhole-trait-export-v1.clar",
      "wormhole-trait-governance-v1.clar",
      "wormhole-core-v4.clar",
    ].map((filename) => path.join(contractPath, filename));

    const versionMap = {
      "executor-dao.clar": 3,
    } as Record<string, number>;

    const contracts = [...dependencyFiles, ...contractFiles].map(
      (filePath) => ({
        name: path.basename(filePath).replace(".clar", ""),
        filename: path.basename(filePath),
        code: rewriteClarity(fs.readFileSync(filePath, "utf8")),
      })
    );

    let nonce = await fetchNonce({
      address: ADDRESS,
      client: { baseUrl: STACKS_API_URL },
    });

    const results = {
      totalContracts: contracts.length,
      successfulDeployments: 0,
      contracts: [] as string[],
      deployedTxIds: [] as string[],
      startingNonce: nonce,
    };

    console.log(
      `Deploying ${contracts.length} contracts starting with nonce ${nonce}`
    );

    for (const contract of contracts) {
      const transaction = await makeContractDeploy({
        contractName: contract.name,
        codeBody: contract.code,
        clarityVersion: versionMap?.[contract.filename] ?? 3,
        senderKey: STACKS_PRIVATE_KEY,
        nonce,
        network: "devnet",
        client: { baseUrl: STACKS_API_URL },
      });

      const response = await broadcastTransaction({
        transaction,
        network: "devnet",
        client: { baseUrl: STACKS_API_URL },
      });

      if (
        "error" in response &&
        response.reason === "ContractAlreadyExists"
        // Allow pre existing contracts only for local testing
      ) {
        console.log(
          `Contract ${contract.name} already exists, skipping deployment`
        );
        results.successfulDeployments++;
        results.contracts.push(contract.name);
        continue;
      } else if ("error" in response) {
        throw new Error(
          `Deploy failed for ${contract.name}: ${response.error} ${response.reason}`
        );
      }

      expect(response.txid).toBeDefined();
      expect(response.txid.length).toBe(64);

      console.log(`Deployed ${contract.name}: ${response.txid}`);

      // Wait for transaction to be successful before proceeding
      await waitForTransactionSuccess(response.txid);

      results.successfulDeployments++;
      results.contracts.push(contract.name);
      results.deployedTxIds.push(response.txid);
      nonce += 1n;
    }

    expect(results.totalContracts).toBeGreaterThan(0);
    expect(results.successfulDeployments).toBe(results.totalContracts);
    expect(results.contracts).toContain("wormhole-core-state");
    expect(results.contracts).toContain("wormhole-core-v4");
    expect(results.contracts).toContain("wormhole-core-proxy-v2");
  });

  it("should initialize stacks core contract", async () => {
    const ADDRESS = privateKeyToAddress(STACKS_PRIVATE_KEY, "devnet");

    const nonce = await fetchNonce({
      address: ADDRESS,
      client: { baseUrl: STACKS_API_URL },
    });

    const transaction = await makeContractCall({
      contractAddress: ADDRESS,
      contractName: "wormhole-core-v4",
      functionName: "initialize",
      functionArgs: [Cl.none()],
      senderKey: STACKS_PRIVATE_KEY,
      fee: 50_000,
      nonce,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    const response = await broadcastTransaction({
      transaction,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    if ("txid" in response && response.txid) {
      console.log(`Initialized core: ${response.txid}`);

      try {
        await waitForTransactionSuccess(response.txid);
      } catch (error) {
        // Check if it's the "already initialized" error (u10003)
        if (
          // Allow already initialized only for local testing
          error instanceof Error &&
          error.message.includes("(err u10003)")
        ) {
          console.log(`Core already initialized, continuing...`);
        } else throw error;
      }

      const owner = await fetchCallReadOnlyFunction({
        contractAddress: ADDRESS,
        contractName: "wormhole-core-state",
        functionName: "get-active-wormhole-core-contract",
        functionArgs: [],
        network: "devnet",
        client: { baseUrl: STACKS_API_URL },
        senderAddress: ADDRESS,
      });

      expect(owner).toBeDefined();
      console.log(`Core initialization verified`);
    } else {
      console.error(`Failed to initialize core:`, response);
      throw new Error(`Core initialization failed`);
    }
  });

  it("should upgrade guardian set", async () => {
    const ADDRESS = privateKeyToAddress(STACKS_PRIVATE_KEY, "devnet");

    const exportedVars = (await fetchCallReadOnlyFunction({
      contractAddress: ADDRESS,
      contractName: "wormhole-core-v4",
      functionName: "get-exported-vars",
      functionArgs: [],
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
      senderAddress: ADDRESS,
    })) as TupleCV<{ "active-guardian-set-id": UIntCV }>;

    const activeGuardianSetId = Number(
      exportedVars.value["active-guardian-set-id"].value
    );

    const keychain = wormhole.generateGuardianSetKeychain(19);
    const guardianSetUpgrade = wormhole.generateGuardianSetUpdateVaa(
      keychain,
      activeGuardianSetId + 1
    );

    const nonce = await fetchNonce({
      address: ADDRESS,
      client: { baseUrl: STACKS_API_URL },
    });

    const transaction = await makeContractCall({
      contractAddress: ADDRESS,
      contractName: "wormhole-core-v4",
      functionName: "guardian-set-upgrade",
      functionArgs: [
        Cl.buffer(guardianSetUpgrade.vaa),
        Cl.list(guardianSetUpgrade.uncompressedPublicKeys),
      ],
      senderKey: STACKS_PRIVATE_KEY,
      fee: 100_000,
      nonce,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    const response = await broadcastTransaction({
      transaction,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    if ("txid" in response && response.txid) {
      console.log(`Guardian set upgrade: ${response.txid}`);

      try {
        await waitForTransactionSuccess(response.txid);

        const guardianSet = await fetchContractMapEntry({
          contractAddress: ADDRESS,
          contractName: "wormhole-core-state",
          mapName: "guardian-sets",
          mapKey: Cl.uint(1),
          network: "devnet",
          client: { baseUrl: STACKS_API_URL },
        });

        expect(guardianSet).toBeDefined();
        console.log(`Guardian set upgrade verified`);
      } catch (error) {
        if (
          // Allow existing guardian set only for local testing
          error instanceof Error &&
          error.message.includes("(err u1102)")
        ) {
          console.log(`Guardian set upgrade failed, continuing...`);
        } else {
          throw error;
        }
      }
    } else {
      console.error(`Failed to upgrade guardian set:`, response);
      throw new Error(`Guardian set upgrade failed`);
    }
  });

  it("should post and spy onmessage", async () => {
    const ADDRESS = privateKeyToAddress(STACKS_PRIVATE_KEY, "devnet");

    const payload = Buffer.from("test-payload-success-case");
    const messageNonce = Math.floor(Math.random() * 0xffffffff);

    const spyPromise = expectVAA(payload);

    const nonce = await fetchNonce({
      address: ADDRESS,
      client: { baseUrl: STACKS_API_URL },
    });

    const transaction = await makeContractCall({
      contractAddress: ADDRESS,
      contractName: "wormhole-core-v4",
      functionName: "post-message",
      functionArgs: [Cl.buffer(payload), Cl.uint(messageNonce), Cl.none()],
      postConditionMode: "allow",
      senderKey: STACKS_PRIVATE_KEY,
      fee: 100_000,
      nonce,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    const response = await broadcastTransaction({
      transaction,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    if ("error" in response) throw new Error(response.error);

    console.log(`Posted message: ${response.txid}`);
    await waitForTransactionSuccess(response.txid);

    expect(response.txid).toBeDefined();
    expect(response.txid.length).toBe(64);

    await expect(spyPromise).resolves.toBeUndefined();
  });

  it("should spy but not find VAA for faulty transaction (abort_by_post_condition)", async () => {
    const ADDRESS = privateKeyToAddress(STACKS_PRIVATE_KEY, "devnet");

    const payload = Buffer.from("test-payload-abort-by-post-condition");
    const messageNonce = Math.floor(Math.random() * 0xffffffff);

    const spyPromise = expectNoStacksVAA();

    const nonce = await fetchNonce({
      address: ADDRESS,
      client: { baseUrl: STACKS_API_URL },
    });

    const transaction = await makeContractCall({
      contractAddress: ADDRESS,
      contractName: "wormhole-core-v4",
      functionName: "post-message",
      functionArgs: [Cl.buffer(payload), Cl.uint(messageNonce), Cl.none()],
      postConditionMode: "allow",
      postConditions: [Pc.origin().willSendEq(66).ustx()],
      senderKey: STACKS_PRIVATE_KEY,
      fee: 100_000,
      nonce,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    const response = await broadcastTransaction({
      transaction,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    if ("error" in response) throw new Error(response.error);

    console.log(
      `Posted faulty message (abort_by_post_condition): ${response.txid}`
    );
    await waitForTransactionSuccess(response.txid);

    expect(response.txid).toBeDefined();
    expect(response.txid.length).toBe(64);

    await expect(spyPromise).resolves.toBeUndefined();
  });

  it("should spy but not find VAA for faulty transaction (abort_by_response) #1", async () => {
    const ADDRESS = privateKeyToAddress(STACKS_PRIVATE_KEY, "devnet");

    const payload = Buffer.from("test-payload-abort-by-response-1");

    const spyPromise = expectNoStacksVAA();

    const nonce = await fetchNonce({
      address: ADDRESS,
      client: { baseUrl: STACKS_API_URL },
    });

    // Deploy a contract that calls post-message on deploy
    // The contract-call succeeds but transaction fails because response isn't handled
    const clarityCode = `(begin
                            (contract-call? '${ADDRESS}.wormhole-core-v4 post-message
                                0x${payload.toString("hex")}
                                u42
                                none)
                            (err u1))`;

    const transaction = await makeContractDeploy({
      contractName: `test-post-message-abort-${nonce}`,
      codeBody: clarityCode,
      clarityVersion: 3,
      senderKey: STACKS_PRIVATE_KEY,
      fee: 100_000,
      nonce,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    const response = await broadcastTransaction({
      transaction,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    if ("error" in response) {
      console.log(`Deploy failed: ${JSON.stringify(response, null, 2)}`);
      throw new Error(response.error);
    }

    console.log(`Posted faulty message (abort_by_response): ${response.txid}`);
    try {
      await waitForTransactionSuccess(response.txid);
    } catch (error) {
      if (error instanceof Error && error.message.includes("(err none)")) {
        console.log(`Transaction failed as expected.`);
      }
    }

    expect(response.txid).toBeDefined();
    expect(response.txid.length).toBe(64);

    await expect(spyPromise).resolves.toBeUndefined();
  });

  it("should spy but not find VAA for faulty transaction (abort_by_response) #2", async () => {
    const ADDRESS = privateKeyToAddress(STACKS_PRIVATE_KEY, "devnet");

    const payload = Buffer.from("test-payload-abort-by-response-2");

    const spyPromise = expectNoStacksVAA();

    let nonce = await fetchNonce({
      address: ADDRESS,
      client: { baseUrl: STACKS_API_URL },
    });

    // STEP 1: Deploy a contract with a public function that calls post-message and returns error
    const clarityCode = `(define-public (post-and-fail)
  (begin
    (try! (contract-call? '${ADDRESS}.wormhole-core-v4 post-message
      0x${payload.toString("hex")}
      u43
      none))
    (err u1)))`;

    const deployTransaction = await makeContractDeploy({
      contractName: `test-post-message-two-step-${nonce}`,
      codeBody: clarityCode,
      clarityVersion: 3,
      senderKey: STACKS_PRIVATE_KEY,
      fee: 100_000,
      nonce,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    const deployResponse = await broadcastTransaction({
      transaction: deployTransaction,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    if ("error" in deployResponse) {
      console.log(`Deploy failed: ${JSON.stringify(deployResponse, null, 2)}`);
      throw new Error(deployResponse.error);
    }

    console.log(`Deployed two-step test contract: ${deployResponse.txid}`);
    try {
      await waitForTransactionSuccess(deployResponse.txid);
    } catch (error) {
      if (error instanceof Error && error.message.includes("(err")) {
        console.log(`Transaction failed as expected.`);
      }
    }

    expect(deployResponse.txid).toBeDefined();
    expect(deployResponse.txid.length).toBe(64);

    nonce += 1n;

    // STEP 2: Call the public function that will post-message and return error
    const callTransaction = await makeContractCall({
      contractAddress: ADDRESS,
      contractName: `test-post-message-two-step-${nonce - 1n}`,
      functionName: "post-and-fail",
      functionArgs: [],
      postConditionMode: "allow",
      senderKey: STACKS_PRIVATE_KEY,
      fee: 100_000,
      nonce,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    const callResponse = await broadcastTransaction({
      transaction: callTransaction,
      network: "devnet",
      client: { baseUrl: STACKS_API_URL },
    });

    if ("error" in callResponse) {
      console.log(`Call failed: ${JSON.stringify(callResponse, null, 2)}`);
      throw new Error(callResponse.error);
    }

    console.log(
      `Posted faulty message (abort_by_response_two_step): ${callResponse.txid}`
    );
    try {
      await waitForTransactionSuccess(callResponse.txid);
    } catch (error) {
      if (error instanceof Error && error.message.includes("(err")) {
        console.log(`Transaction failed as expected.`);
      }
    }

    expect(callResponse.txid).toBeDefined();
    expect(callResponse.txid.length).toBe(64);

    await expect(spyPromise).resolves.toBeUndefined();
  });
});
