"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.fromBase64 = exports.toValAddress = exports.toBech32 = exports.fromValAddress = exports.fromBech32 = exports.unpackHttpReponse = exports.getGuardianValidatorRegistrations = exports.getValidators = exports.getActiveGuardianSet = exports.getGuardianSets = exports.executeGovernanceVAA = exports.getAddress = exports.getWallet = exports.getZeroFee = exports.getStargateClient = exports.getStargateQueryClient = exports.LCD_URL = exports.HOLE_DENOM = exports.TENDERMINT_URL = void 0;
var fetch = require("node-fetch");
//@ts-ignore
globalThis.fetch = fetch;
const bech32_1 = require("bech32");
const proto_signing_1 = require("@cosmjs/proto-signing");
const stargate_1 = require("@cosmjs/stargate");
const tendermint_rpc_1 = require("@cosmjs/tendermint-rpc");
const certusone_wormholechain_wormhole_1 = require("./modules/certusone.wormholechain.wormhole");
//https://tutorials.cosmos.network/academy/4-my-own-chain/cosmjs.html
const ADDRESS_PREFIX = "wormhole";
const OPERATOR_PREFIX = "wormholevaloper";
exports.TENDERMINT_URL = "http://localhost:26657";
exports.HOLE_DENOM = "uhole";
exports.LCD_URL = "http://localhost:1317";
async function getStargateQueryClient() {
    const tmClient = await tendermint_rpc_1.Tendermint34Client.connect(exports.TENDERMINT_URL);
    const client = stargate_1.QueryClient.withExtensions(tmClient, stargate_1.setupTxExtension, stargate_1.setupGovExtension, stargate_1.setupIbcExtension, stargate_1.setupAuthExtension, stargate_1.setupBankExtension, stargate_1.setupMintExtension, stargate_1.setupStakingExtension);
    return client;
}
exports.getStargateQueryClient = getStargateQueryClient;
async function getStargateClient() {
    const client = await stargate_1.StargateClient.connect(exports.TENDERMINT_URL);
    return client;
}
exports.getStargateClient = getStargateClient;
function getZeroFee() {
    return {
        amount: (0, proto_signing_1.coins)(0, exports.HOLE_DENOM),
        gas: "180000", // 180k",
    };
}
exports.getZeroFee = getZeroFee;
async function getWallet(mnemonic) {
    const wallet = await proto_signing_1.DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
        prefix: ADDRESS_PREFIX,
    });
    return wallet;
}
exports.getWallet = getWallet;
async function getAddress(wallet) {
    //There are actually up to 5 accounts in a cosmos wallet. I believe this returns the first wallet.
    const [{ address }] = await wallet.getAccounts();
    return address;
}
exports.getAddress = getAddress;
async function executeGovernanceVAA(wallet, hexVaa) {
    const offline = wallet;
    const client = await (0, certusone_wormholechain_wormhole_1.txClient)(offline);
    const msg = client.msgExecuteGovernanceVAA({
        vaa: new Uint8Array(),
        signer: await getAddress(wallet),
    }); //TODO convert type
    const signingClient = await stargate_1.SigningStargateClient.connectWithSigner(exports.TENDERMINT_URL, wallet
    //{ gasPrice: { amount: Decimal.fromUserInput("0.0", 0), denom: "uhole" } }
    );
    //TODO investigate signing with the stargate client, as the module txClients can't do 100% of the operations
    //   const output = signingClient.signAndBroadcast(
    //     await getAddress(wallet),
    //     [msg],
    //     getZeroFee(),
    //     "executing governance VAA"
    //   );
    //TODO the EncodingObjects from the txClient seem to be incompatible with the
    //stargate client
    // In order for all the encoding objects to be interoperable, we will have to either coerce the txClient msgs into the format of stargate,
    // or we could just just txClients for everything. I am currently leaning towards the latter, as we can generate txClients for everything out of the cosmos-sdk,
    // and we will likely need to generate txClients for our forked version of the cosmos SDK anyway.
    const output = await client.signAndBroadcast([msg]);
    return output;
}
exports.executeGovernanceVAA = executeGovernanceVAA;
async function getGuardianSets() {
    const client = await (0, certusone_wormholechain_wormhole_1.queryClient)({ addr: exports.LCD_URL });
    const response = client.queryGuardianSetAll();
    return await unpackHttpReponse(response);
}
exports.getGuardianSets = getGuardianSets;
async function getActiveGuardianSet() {
    const client = await (0, certusone_wormholechain_wormhole_1.queryClient)({ addr: exports.LCD_URL });
    const response = client.queryActiveGuardianSetIndex();
    return await unpackHttpReponse(response);
}
exports.getActiveGuardianSet = getActiveGuardianSet;
async function getValidators() {
    const client = await getStargateQueryClient();
    //TODO handle pagination here
    const validators = await client.staking.validators("BOND_STATUS_BONDED");
    return validators;
}
exports.getValidators = getValidators;
async function getGuardianValidatorRegistrations() {
    const client = await (0, certusone_wormholechain_wormhole_1.queryClient)({ addr: exports.LCD_URL });
    const response = client.queryGuardianValidatorAll();
    return await unpackHttpReponse(response);
}
exports.getGuardianValidatorRegistrations = getGuardianValidatorRegistrations;
async function unpackHttpReponse(response) {
    const http = await response;
    //TODO check rpc status
    const content = http.data;
    return content;
}
exports.unpackHttpReponse = unpackHttpReponse;
function fromBech32(address) {
    return Buffer.from(bech32_1.bech32.decode(address).words);
}
exports.fromBech32 = fromBech32;
function fromValAddress(valAddress) {
    return Buffer.from(bech32_1.bech32.decode(valAddress).words);
}
exports.fromValAddress = fromValAddress;
function toBech32(address) {
    return bech32_1.bech32.encode(ADDRESS_PREFIX, address);
}
exports.toBech32 = toBech32;
function toValAddress(address) {
    return bech32_1.bech32.encode(OPERATOR_PREFIX, address);
}
exports.toValAddress = toValAddress;
function fromBase64(address) {
    return Buffer.from(bech32_1.bech32.toWords(Buffer.from(address, "base64")));
}
exports.fromBase64 = fromBase64;
