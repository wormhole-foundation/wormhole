"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
var wormhole_sdk_1 = require("@certusone/wormhole-sdk");
var wasm_1 = require("@certusone/wormhole-sdk/lib/cjs/solana/wasm");
var address_1 = require("@ethersproject/address");
var units_1 = require("@ethersproject/units");
var ethers_1 = require("ethers");
var fs = require("fs");
var process_1 = require("process");
var SimpleDex__factory_1 = require("../ethers-contracts/abi/factories/SimpleDex__factory");
var consts_1 = require("./consts");
var commonWorkflows_1 = require("@certusone/wormhole-examples/lib/commonWorkflows");
wasm_1.setDefaultWasm("node");
//This script is reliant on core examples, and the wormhole SDK.
//It is meant to be run against a fresh devnet / tilt environment.
function getPrice(chain) {
    if (chain === wormhole_sdk_1.CHAIN_ID_ETH) {
        return 4400;
    }
    if (chain === wormhole_sdk_1.CHAIN_ID_BSC) {
        return 630;
    }
}
function main() {
    return __awaiter(this, void 0, void 0, function () {
        var _a, ethAddress, wbnbOnEth, bscAddress, wethOnBsc;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0: return [4 /*yield*/, configWormhole()];
                case 1:
                    _b.sent();
                    return [4 /*yield*/, seedPools()];
                case 2:
                    _a = _b.sent(), ethAddress = _a.ethAddress, wbnbOnEth = _a.wbnbOnEth, bscAddress = _a.bscAddress, wethOnBsc = _a.wethOnBsc;
                    console.log("Pools seeded");
                    return [4 /*yield*/, createSwapPoolFile(ethAddress, wbnbOnEth, bscAddress, wethOnBsc)];
                case 3:
                    _b.sent();
                    console.log("Job done");
                    return [2 /*return*/, Promise.resolve()];
            }
        });
    });
}
function configWormhole() {
    return __awaiter(this, void 0, void 0, function () {
        var basisTransferAmount, WETH;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    basisTransferAmount = "1";
                    WETH = "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
                    console.log("Doing WETH Attest");
                    return [4 /*yield*/, commonWorkflows_1.fullAttestation(wormhole_sdk_1.CHAIN_ID_ETH, address_1.getAddress(WETH))];
                case 1:
                    _a.sent();
                    console.log("Doing WBNB Attest");
                    return [4 /*yield*/, commonWorkflows_1.fullAttestation(wormhole_sdk_1.CHAIN_ID_BSC, address_1.getAddress(WETH))];
                case 2:
                    _a.sent();
                    console.log("Bridging over WETH to bsc");
                    return [4 /*yield*/, commonWorkflows_1.basicTransfer(wormhole_sdk_1.CHAIN_ID_ETH, basisTransferAmount, wormhole_sdk_1.CHAIN_ID_BSC, consts_1.ETH_TEST_WALLET_PUBLIC_KEY, consts_1.ETH_TEST_WALLET_PUBLIC_KEY, true)];
                case 3:
                    _a.sent();
                    console.log("Bridging over WBNB to eth");
                    return [4 /*yield*/, commonWorkflows_1.basicTransfer(wormhole_sdk_1.CHAIN_ID_BSC, basisTransferAmount, wormhole_sdk_1.CHAIN_ID_ETH, consts_1.ETH_TEST_WALLET_PUBLIC_KEY, consts_1.ETH_TEST_WALLET_PUBLIC_KEY, true)];
                case 4:
                    _a.sent();
                    return [2 /*return*/];
            }
        });
    });
}
function createSwapPoolFile(ethAddress, wbnbAddress, bscAddress, wethAddress) {
    return __awaiter(this, void 0, void 0, function () {
        var literal, content;
        var _a, _b, _c;
        return __generator(this, function (_d) {
            switch (_d.label) {
                case 0:
                    literal = (_a = {},
                        _a[wormhole_sdk_1.CHAIN_ID_ETH] = (_b = {},
                            _b[wormhole_sdk_1.CHAIN_ID_BSC] = { poolAddress: ethAddress, tokenAddress: wbnbAddress },
                            _b),
                        _a[wormhole_sdk_1.CHAIN_ID_BSC] = (_c = {},
                            _c[wormhole_sdk_1.CHAIN_ID_ETH] = { poolAddress: bscAddress, tokenAddress: wethAddress },
                            _c),
                        _a);
                    content = JSON.stringify(literal);
                    //TODO not this
                    return [4 /*yield*/, fs.writeFileSync("../react/src/swapPools.json", content, {
                            flag: "w+",
                        })];
                case 1:
                    //TODO not this
                    _d.sent();
                    return [2 /*return*/];
            }
        });
    });
}
//TODO, in a for loop for all the EVM chains
var seedPools = function () { return __awaiter(void 0, void 0, void 0, function () {
    var ethSigner, bscSigner, currentEthPrice, currentSolPrice, ratio, ethBasis, bnbBasis, WETH, wbnbOnEth, wethOnBsc, contractInterface, bytecode, ethfactory, contract, ethAddress, bscfactory, bscContract, bscAddress, ethDex, bscDex, ethInit, bscInit, ethLiq, bscLiq;
    return __generator(this, function (_a) {
        switch (_a.label) {
            case 0:
                ethSigner = consts_1.getSignerForChain(wormhole_sdk_1.CHAIN_ID_ETH);
                bscSigner = consts_1.getSignerForChain(wormhole_sdk_1.CHAIN_ID_BSC);
                currentEthPrice = getPrice(wormhole_sdk_1.CHAIN_ID_ETH);
                currentSolPrice = getPrice(wormhole_sdk_1.CHAIN_ID_BSC);
                ratio = currentEthPrice / currentSolPrice;
                ethBasis = 0.01;
                bnbBasis = Math.ceil(ethBasis * ratio);
                WETH = "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
                return [4 /*yield*/, wormhole_sdk_1.getForeignAssetEth(consts_1.getTokenBridgeAddressForChain(wormhole_sdk_1.CHAIN_ID_ETH), ethSigner.provider, wormhole_sdk_1.CHAIN_ID_BSC, wormhole_sdk_1.hexToUint8Array(wormhole_sdk_1.nativeToHexString(WETH, wormhole_sdk_1.CHAIN_ID_BSC)))];
            case 1:
                wbnbOnEth = _a.sent();
                console.log("WBNB on ETH address", wbnbOnEth);
                return [4 /*yield*/, wormhole_sdk_1.getForeignAssetEth(consts_1.getTokenBridgeAddressForChain(wormhole_sdk_1.CHAIN_ID_BSC), bscSigner.provider, wormhole_sdk_1.CHAIN_ID_ETH, wormhole_sdk_1.hexToUint8Array(wormhole_sdk_1.nativeToHexString(WETH, wormhole_sdk_1.CHAIN_ID_ETH)))];
            case 2:
                wethOnBsc = _a.sent();
                console.log("WETH on BSC address", wethOnBsc);
                console.log("about to deploy ETH contract");
                contractInterface = SimpleDex__factory_1.SimpleDex__factory.createInterface();
                bytecode = SimpleDex__factory_1.SimpleDex__factory.bytecode;
                ethfactory = new ethers_1.ethers.ContractFactory(contractInterface, bytecode, ethSigner);
                return [4 /*yield*/, ethfactory.deploy(address_1.getAddress(wbnbOnEth))];
            case 3:
                contract = _a.sent();
                return [4 /*yield*/, contract.deployed().then(function (result) {
                        console.log("Successfully deployed contract at " + result.address);
                        return result.address;
                    }, function (error) {
                        console.error(error);
                        process_1.exit(1);
                    })];
            case 4:
                ethAddress = _a.sent();
                console.log("about to deploy bsc contract");
                bscfactory = new ethers_1.ethers.ContractFactory(contractInterface, bytecode, bscSigner);
                return [4 /*yield*/, bscfactory.deploy(address_1.getAddress(wethOnBsc))];
            case 5:
                bscContract = _a.sent();
                return [4 /*yield*/, bscContract.deployed().then(function (result) {
                        console.log("Successfully deployed contract at " + result.address);
                        return result.address;
                    }, function (error) {
                        console.error(error);
                        process_1.exit(1);
                    })];
            case 6:
                bscAddress = _a.sent();
                console.log("Doing WBNB on ETH Approve");
                return [4 /*yield*/, wormhole_sdk_1.approveEth(ethAddress, address_1.getAddress(wbnbOnEth), ethSigner, "10000000000000000000000")];
            case 7:
                _a.sent();
                console.log("Doing WETH on BSC Approve");
                return [4 /*yield*/, wormhole_sdk_1.approveEth(bscAddress, address_1.getAddress(wethOnBsc), bscSigner, "10000000000000000000000")];
            case 8:
                _a.sent();
                ethDex = SimpleDex__factory_1.SimpleDex__factory.connect(ethAddress, ethSigner);
                bscDex = SimpleDex__factory_1.SimpleDex__factory.connect(bscAddress, bscSigner);
                console.log("Initializing eth pool");
                return [4 /*yield*/, ethDex.init(units_1.parseUnits(bnbBasis.toString(), 18), {
                        value: units_1.parseUnits(ethBasis.toString(), 18),
                        gasLimit: 500000,
                    })];
            case 9:
                ethInit = _a.sent();
                return [4 /*yield*/, ethInit.wait()];
            case 10:
                _a.sent();
                console.log("Initializing bsc pool");
                return [4 /*yield*/, bscDex.init(units_1.parseUnits(ethBasis.toString(), 18), {
                        value: units_1.parseUnits(bnbBasis.toString(), 18),
                        gasLimit: 500000,
                    })];
            case 11:
                bscInit = _a.sent();
                return [4 /*yield*/, bscInit.wait()];
            case 12:
                _a.sent();
                console.log("pools initialized");
                return [4 /*yield*/, ethDex.totalLiquidity()];
            case 13:
                ethLiq = _a.sent();
                console.log("Eth liquidity", ethLiq);
                return [4 /*yield*/, bscDex.totalLiquidity()];
            case 14:
                bscLiq = _a.sent();
                console.log("bsc liquidity", bscLiq);
                //Pool should now be seeded with a small amount of ETH and Wormhole-Wrapped SOL.
                return [2 /*return*/, { ethAddress: ethAddress, wbnbOnEth: wbnbOnEth, bscAddress: bscAddress, wethOnBsc: wethOnBsc }];
        }
    });
}); };
var done = false;
main().then(function () { return (done = true); }, function (error) {
    console.error(error);
    done = true;
});
function wait() {
    if (!done) {
        setTimeout(wait, 1000);
    }
    else {
        process_1.exit(0);
    }
}
wait();
