import { execSync } from "child_process";
import { Bech32, toHex } from "@cosmjs/encoding";
import {
  makeTransferVaaPayload,
  signAndEncodeVaa,
  TEST_SIGNER_PKS,
} from "./src/helpers/vaa";
import { keccak256 } from "ethers/lib/utils";

function sleep(ms: number) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

function createHashQueryString(hash: string) {
  const txhashQuery = "injectived query tx ";
  const jsonFlag = " -o json";
  return txhashQuery + hash + jsonFlag;
}

function nativeToHex(address: string) {
  return toHex(Bech32.decode(address).data).padStart(64, "0");
}

// wasm contracts
const wormholeWasm: string = "wormhole.wasm";
const tokenBridgeWasm: string = "token_bridge_terra_2.wasm";
const cw20WrappedWasm: string = "cw20_wrapped_2.wasm";
const mockBridgeWasm: string = "mock_bridge_integration_2.wasm";

function createWasmFullPath(w: string) {
  const pathToWasms: string =
    "/home/pnoel/git/injective/wormhole/cosmwasm/artifacts/";
  return pathToWasms + w;
}

console.log("About to execute Injective commands...");

executeCommands();

async function executeCommands() {
  let GenesisContract = "";
  let WormholeCodeId = "";
  const storeWormhole: string =
    "yes 12345678 | injectived tx wasm store " +
    createWasmFullPath(wormholeWasm) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1500000000000000inj --gas=3000000 --output=JSON';
  // Store the wormhole.wasm contract on chain
  let esResult = execSync(storeWormhole);
  let out: string = esResult.toString();
  // console.log("rinj", out);
  let parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Get the raw log to look for success (and code_id)
  await sleep(4000);
  let txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
  const attr = parsed[0]["events"][0].attributes;
  // console.log("raw_log as json:", attr);
  attr.forEach((element) => {
    // console.log("element", element);
    if (element.key === "sender") {
      // console.log("Got sender.  Look for genesis contract.");
      GenesisContract = element.value;
    }
  });
  if (GenesisContract === "") {
    console.error("Could not parse genesis address.");
    return;
  }
  console.log("Genesis contract address: ", GenesisContract);
  let storeCode = parsed[0]["events"][1].attributes;
  // console.log("storeCode", storeCode);
  storeCode.forEach((element) => {
    // console.log("element", element);
    if (element.key === "code_id") {
      // console.log("Got sender.  Look for genesis contract.");
      WormholeCodeId = element.value;
    }
  });
  if (WormholeCodeId === "") {
    console.error("Could not parse wormhole contract code id.");
    return;
  }
  console.log("Wormhole contract code id: ", WormholeCodeId);

  // Instantiate the wormhole contract
  const instantiateWormhole: string =
    "yes 12345678 | injectived tx wasm instantiate " +
    WormholeCodeId +
    " " +
    JSON.stringify(
      '{"gov_chain": 1, "gov_address": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=", "guardian_set_expirity": 86400, "initial_guardian_set": {"addresses": [{"bytes": "vvpCnVfNGLf4pNkaLamrSvBdD74="}], "expiration_time": 0}}'
    ) +
    ' --label="Wormhole" --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --no-admin --output=JSON';
  console.log("about to execute:", instantiateWormhole);
  esResult = execSync(instantiateWormhole);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash to get the wormhole contract address
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
  let WormholeContractAddress = "";
  let attrs = parsed[0]["events"];
  // console.log("raw_log as json:", attrs);
  attrs.forEach((element) => {
    const elAttrs = element.attributes[0];
    // console.log("element", elAttrs);
    if (elAttrs.key === "_contract_address") {
      // console.log("Got _contract_address.");
      WormholeContractAddress = elAttrs.value;
    }
  });
  if (WormholeContractAddress === "") {
    console.error("Could not parse wormhole contract address.");
    return;
  }
  console.log("Wormhole contract address: ", WormholeContractAddress);

  // Load the Token Bridge wasm contract
  const storeTokenBridge: string =
    "yes 12345678 | injectived tx wasm store " +
    createWasmFullPath(tokenBridgeWasm) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1500000000000000inj --gas=3000000 --output=JSON';
  let TokenBridgeCodeId = "";
  console.log("about to execute:", storeTokenBridge);
  esResult = execSync(storeTokenBridge);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Get the raw log to look for success (and code_id)
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
  storeCode = parsed[0]["events"][1].attributes;
  // console.log("storeCode", storeCode);
  storeCode.forEach((element) => {
    // console.log("element", element);
    if (element.key === "code_id") {
      // console.log("Got code_id.");
      TokenBridgeCodeId = element.value;
    }
  });
  if (TokenBridgeCodeId === "") {
    console.error("Could not parse token bridge contract code id.");
    return;
  }
  console.log("TokenBridge contract code id: ", TokenBridgeCodeId);

  // Load the CW20 token wasm contract
  const storeCW20: string =
    "yes 12345678 | injectived tx wasm store " +
    createWasmFullPath(cw20WrappedWasm) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1500000000000000inj --gas=3000000 --output=JSON';
  let CW20TokenId = "";
  console.log("about to execute:", storeCW20);
  esResult = execSync(storeCW20);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Get the raw log to look for success (and code_id)
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
  storeCode = parsed[0]["events"][1].attributes;
  // console.log("storeCode", storeCode);
  storeCode.forEach((element) => {
    // console.log("element", element);
    if (element.key === "code_id") {
      // console.log("Got code_id.");
      CW20TokenId = element.value;
    }
  });
  if (CW20TokenId === "") {
    console.error("Could not parse token bridge contract code id.");
    return;
  }
  console.log("CW20 token contract code id: ", CW20TokenId);

  // Instantiate the token bridge
  const tbInstJson =
    '{"gov_chain":1, "gov_address":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=", "wormhole_contract":"' +
    WormholeContractAddress +
    '", "wrapped_asset_code_id":' +
    CW20TokenId +
    "}";
  const instantiateTokenBridge: string =
    "yes 12345678 | injectived tx wasm instantiate " +
    TokenBridgeCodeId +
    " " +
    JSON.stringify(tbInstJson) +
    ' --label="tokenBridge" --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --no-admin --output=JSON';
  console.log("about to execute:", instantiateTokenBridge);
  esResult = execSync(instantiateTokenBridge);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash to get the token bridge contract address
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
  let TokenBridgeContractAddress = "";
  attrs = parsed[0]["events"];
  // console.log("raw_log as json:", attrs);
  attrs.forEach((element) => {
    const elAttrs = element.attributes[0];
    // console.log("element", elAttrs);
    if (elAttrs.key === "_contract_address") {
      // console.log("Got _contract_address.");
      TokenBridgeContractAddress = elAttrs.value;
    }
  });
  if (TokenBridgeContractAddress === "") {
    console.error("Could not parse wormhole contract address.");
    return;
  }
  console.log("Token Bridge contract address: ", TokenBridgeContractAddress);

  // Load the Mock Bridge wasm contract
  const storeMockBridge: string =
    "yes 12345678 | injectived tx wasm store " +
    createWasmFullPath(mockBridgeWasm) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1500000000000000inj --gas=3000000 --output=JSON';
  let MockBridgeId = "";
  console.log("about to execute:", storeMockBridge);
  esResult = execSync(storeMockBridge);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Get the raw log to look for success (and code_id)
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
  storeCode = parsed[0]["events"][1].attributes;
  // console.log("storeCode", storeCode);
  storeCode.forEach((element) => {
    // console.log("element", element);
    if (element.key === "code_id") {
      // console.log("Got code_id.");
      MockBridgeId = element.value;
    }
  });
  if (MockBridgeId === "") {
    console.error("Could not parse mock bridge contract code id.");
    return;
  }
  console.log("Mock bridge contract code id: ", MockBridgeId);

  // Instantiate the mock bridge
  const mbInstJson =
    '{"token_bridge_contract":"' + TokenBridgeContractAddress + '"}';
  const instantiateMockBridge: string =
    "yes 12345678 | injectived tx wasm instantiate " +
    MockBridgeId +
    " " +
    JSON.stringify(mbInstJson) +
    ' --label="mockBridgeIntegration" --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --no-admin --output=JSON';
  console.log("about to execute:", instantiateMockBridge);
  esResult = execSync(instantiateMockBridge);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash to get the mock bridge contract address
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
  let MockBridgeContractAddress = "";
  attrs = parsed[0]["events"];
  // console.log("raw_log as json:", attrs);
  attrs.forEach((element) => {
    const elAttrs = element.attributes[0];
    // console.log("element", elAttrs);
    if (elAttrs.key === "_contract_address") {
      // console.log("Got _contract_address.");
      MockBridgeContractAddress = elAttrs.value;
    }
  });
  if (MockBridgeContractAddress === "") {
    console.error("Could not parse wormhole contract address.");
    return;
  }
  console.log("Mock Bridge contract address: ", MockBridgeContractAddress);

  // Register a foreign bridge via submit_vaa
  console.log("         Register a foreign bridge...");
  const ForeignBridgeSignedVaa: string =
    "0100000000010015fb760dfb7014b7013c7b3a2d9746edb33050076375a76682c0fb49517986844a7fee81094e726f642fa1bc6c52d30ed8f1d5fc22e851d1a9ee8e757b5cfbff01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000001000000000000000000000000000000000000000000000000000000000000ffff";
  const fbExecJson =
    '{"submit_vaa":{"data":"' +
    Buffer.from(ForeignBridgeSignedVaa, "hex").toString("base64") +
    '"}}';
  const execForeignBridge: string =
    "yes 12345678 | injectived tx wasm execute " +
    TokenBridgeContractAddress +
    " " +
    JSON.stringify(fbExecJson) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=JSON';
  console.log("about to execute:", execForeignBridge);
  esResult = execSync(execForeignBridge);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);

  // Deposit native tokens
  console.log("         Deposit native tokens...");
  const depNatJson = '{ "deposit_tokens": {}}';
  const execDepositNative: string =
    "yes 12345678 | injectived tx wasm execute " +
    TokenBridgeContractAddress +
    " " +
    JSON.stringify(depNatJson) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --amount 100000000000000000inj --output=JSON';
  console.log("about to execute:", execDepositNative);
  esResult = execSync(execDepositNative);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);

  // TODO:  Add injectived query bank balances TokenBridgeContractAddress?

  // Initiate a transfer of the deposit that was just made.
  console.log("         Initiate transfer...");
  const initXferJson =
    '{"initiate_transfer":{"asset":{"amount":"1000000000000000", "info":{"native_token":{"denom":"inj"}}},"recipient_chain":2,"recipient":"AAAAAAAAAAAAAAAAQgaUIGlCBpQgaUIGlCBpQgaUIGk=","fee":"1000000","nonce":69}}';
  const execInitiateTransfer: string =
    "yes 12345678 | injectived tx wasm execute " +
    TokenBridgeContractAddress +
    " " +
    JSON.stringify(initXferJson) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=JSON';
  console.log("about to execute:", execInitiateTransfer);
  esResult = execSync(execInitiateTransfer);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);

  // Complete a transfer of native tokens via submit_vaa
  console.log("         Complete transfer...");
  const CompleteTransferVaa: string =
    // "01000000000100cae2762576d7bd6fed6f04ddd1322747221b19b6370c13dd19a36688b56e17a95c3d1828435dea743de5c9f8b93f4cabd5d74a5273c358c863a06f52be31b4120000000000000000000001000000000000000000000000000000000000000000000000000000000000ffff000000000000000000010000000000000000000000000000000000000000000000000000000005f5e100017038850bf3af746c36803cce35009268f00d22ae2b55ffb59ac5f2a6add40b001200000000000000000000000047fd63895d8992e5d92dfb00e395516bcd575942001200000000000000000000000000000000000000000000000000000000000f4240";
    "010000000001003e81f06c88c451eb8724bfaeab0533e9eb96a7b598740e2eb9741383ede6427332af25912f80442b91c41f71cb5c66de41f693eee2a6b590e46207c3a6dee5eb0100000000000000000001000000000000000000000000000000000000000000000000000000000000ffff000000000000000000010000000000000000000000000000000000000000000000000000000005f5e100017038850bf3af746c36803cce35009268f00d22ae2b55ffb59ac5f2a6add40b0012000000000000000000000000f7f7dde848e7450a029cd0a9bd9bdae4b5147db3001200000000000000000000000000000000000000000000000000000000000f4240";
  const ctExecJson =
    '{"submit_vaa":{"data":"' +
    Buffer.from(CompleteTransferVaa, "hex").toString("base64") +
    '"}}';
  const execCompleteTransfer: string =
    "yes 12345678 | injectived tx wasm execute " +
    TokenBridgeContractAddress +
    " " +
    JSON.stringify(ctExecJson) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=JSON';
  console.log("about to execute:", execCompleteTransfer);
  esResult = execSync(execCompleteTransfer);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);

  // Initiate transfer with payload
  console.log("         Initiate transfer with payload...");
  // const NativeToHexMockAddr =
  // "0000000000000000000000000736a58fd7bd49e1f059768f8d57649670f68150";
  // This is the mock bridge contract
  const NativeToHexMockAddr = nativeToHex(
    "inj1qum2tr7hh4y7ruzew68c64myjec0dq2s50064k"
  );

  const RecipAddr = Buffer.from(NativeToHexMockAddr, "hex").toString("base64");
  const recipientAddress =
    "0000000000000000000000004206942069420694206942069420694206942069";
  const RecipAddr2 = Buffer.from(recipientAddress, "hex").toString("base64");
  const initXferPayJson =
    '{"initiate_transfer_with_payload":{"asset":{"amount":"1000000000000000", "info":{"native_token":{"denom":"inj"}}},"recipient_chain":2,"recipient":"' +
    RecipAddr2 +
    '","fee":"1000000","payload":"qw==","nonce":69}}';
  const execInitiateTransferPayload: string =
    "yes 12345678 | injectived tx wasm execute " +
    TokenBridgeContractAddress +
    " " +
    JSON.stringify(initXferPayJson) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=JSON';
  console.log("about to execute:", execInitiateTransferPayload);
  esResult = execSync(execInitiateTransferPayload);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);

  // Attest a Native Asset
  console.log("         Attest a Native Asset...");
  const attestJson =
    '{"create_asset_meta":{"asset_info":{"native_token":{"denom":"inj"}},"nonce":69}}';
  const execAttest: string =
    "yes 12345678 | injectived tx wasm execute " +
    TokenBridgeContractAddress +
    " " +
    JSON.stringify(attestJson) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=JSON';
  console.log("about to execute:", execAttest);
  esResult = execSync(execAttest);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);

  // Complete transfer with payload
  console.log("         Complete transfer with payload...");
  const INJ_ADDRESS =
    "01" + keccak256(Buffer.from("inj", "utf-8")).substring(4); // cut off 0x56 (0x prefix - 1 byte)
  const amount = "100000000"; // one benjamin
  const relayerFee = "1000000"; // one dolla

  const encodedTo = nativeToHex(MockBridgeContractAddress);
  const additionalPayload = "All your base are belong to us";

  const vaaPayload = makeTransferVaaPayload(
    3,
    amount,
    INJ_ADDRESS,
    encodedTo,
    18,
    relayerFee, // now sender_address
    additionalPayload
  );

  const timestamp = 1;
  const nonce = 1;
  const sequence = 2;
  const FOREIGN_CHAIN = 1;
  const FOREIGN_TOKEN_BRIDGE =
    "000000000000000000000000000000000000000000000000000000000000ffff";
  const INJ_SIGNER_PKS = ["inj16vmlxdf6mzj4n278qneymd892fzxlqylcqrf4p"]; // Genesis contract

  const signedVaa = signAndEncodeVaa(
    timestamp,
    nonce,
    FOREIGN_CHAIN,
    FOREIGN_TOKEN_BRIDGE,
    sequence,
    vaaPayload,
    TEST_SIGNER_PKS,
    0,
    0
  );

  console.log("Transfer with payload VAA:", signedVaa);
  const completeTransferVaa =
    // "01000000000100195fe34f088d6177e359b0012cba1ef7c0bff6ea35e0ff45c2c5f12c6dd6a1e07dd2a11a1bb63407d9e152068c9e2db9cf3d74e7fd1e7f77537d3a817d2d9cf30100000001000000010001000000000000000000000000000000000000000000000000000000000000ffff000000000000000200030000000000000000000000000000000000000000000000000000000005f5e10001fa6c6fbc36d8c245b0a852a43eb5d644e8b4c477b27bfab9537c10945939da00121399a4e782b935d2bb36b97586d3df8747b07dc66902d807eed0ae99e00ed256001200000000000000000000000000000000000000000000000000000000000f4240416c6c20796f75722062617365206172652062656c6f6e6720746f207573";
    "0100000000010055719230b7c11440294a7cf93929bba54f6c07391c97aff1ace2196643a095fd6a4fb15720cd13c81c0fe6fa65a34139c04d8fdafbee983e039c75f1669944e90000000001000000010001000000000000000000000000000000000000000000000000000000000000ffff000000000000000200030000000000000000000000000000000000000000000000000000000005f5e100017038850bf3af746c36803cce35009268f00d22ae2b55ffb59ac5f2a6add40b00120000000000000000000000000736a58fd7bd49e1f059768f8d57649670f68150001200000000000000000000000000000000000000000000000000000000000f4240416c6c20796f75722062617365206172652062656c6f6e6720746f207573";
  const compXferPayJson =
    '{"complete_transfer_with_payload":{"data":"' +
    Buffer.from(signedVaa, "hex").toString("base64") +
    '","relayer":"' +
    GenesisContract +
    '"}}';
  const compInitiateTransferPayload: string =
    "yes 12345678 | injectived tx wasm execute " +
    MockBridgeContractAddress +
    " " +
    JSON.stringify(compXferPayJson) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=JSON';
  console.log("about to execute:", compInitiateTransferPayload);
  esResult = execSync(compInitiateTransferPayload);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);

  // Instantiate the CW20 contract
  const cw20InstJson =
    '{ "name": "MOCK", "symbol": "MCK", "decimals": 6, "initial_balances": [ { "address": "' +
    GenesisContract +
    '", "amount": "100000000" } ], "mint": null }';
  console.log("cw20InstJson:", cw20InstJson);
  const instantiateCW20: string =
    "yes 12345678 | injectived tx wasm instantiate " +
    CW20TokenId +
    " " +
    JSON.stringify(cw20InstJson) +
    ' --label="mock" --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --no-admin --output=JSON';
  console.log("about to execute:", instantiateCW20);
  esResult = execSync(instantiateCW20);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash to get the mock bridge contract address
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
  let CW20ContractAddress = "";
  attrs = parsed[0]["events"];
  // console.log("raw_log as json:", attrs);
  attrs.forEach((element) => {
    const elAttrs = element.attributes[0];
    // console.log("element", elAttrs);
    if (elAttrs.key === "_contract_address") {
      // console.log("Got _contract_address.");
      CW20ContractAddress = elAttrs.value;
    }
  });
  if (CW20ContractAddress === "") {
    console.error("Could not parse CW20 contract address.");
    return;
  }
  console.log("CW20 contract address: ", CW20ContractAddress);

  // Attest a CW20 Asset
  console.log("         Attest a CW20 Asset...");
  const attestCW20Json =
    '{"create_asset_meta":{"asset_info":{"token":{"contract_addr":"' +
    CW20ContractAddress +
    '"}},"nonce":69}}';
  const execCW20Attest: string =
    "yes 12345678 | injectived tx wasm execute " +
    TokenBridgeContractAddress +
    " " +
    JSON.stringify(attestCW20Json) +
    ' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=JSON';
  console.log("about to execute:", execCW20Attest);
  esResult = execSync(execCW20Attest);
  out = esResult.toString();
  console.log("rinj", out);
  parsed = JSON.parse(out);
  console.log("parsed", parsed.txhash);

  // Parse the txhash
  await sleep(4000);
  txhashQuery = createHashQueryString(parsed.txhash);
  console.log("About to exec:", txhashQuery);
  esResult = execSync(txhashQuery);
  parsed = JSON.parse(esResult.toString());
  console.log("txhash query raw_log:", parsed.raw_log);
  parsed = JSON.parse(parsed.raw_log);
}
