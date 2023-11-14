import { expectAssignable, expectType } from 'tsd';
import { Chain, Network, RoArray, constMap } from "../../src";



const sample = [
    [
        "Mainnet", [
            ["Ethereum", 1n],
            ["Bsc", 56n],
        ]
    ],
    [
        "Testnet", [
            ["Ethereum", 5n],
            ["Sepolia", 11155111n],
        ]
    ]
] as const satisfies RoArray<readonly [Network, RoArray<readonly [Chain, bigint]>]>;

const test1 = constMap(sample);
const test1Entry1 = test1("Testnet", "Sepolia");
expectAssignable<bigint>(test1Entry1)

const test2 = constMap(sample, [[0, 1], 2]); //same as test1
const test2Entry1 = test2("Testnet", "Sepolia");
expectType<11155111n>(test2Entry1)

// Maps to [1n|5n]
//const test2Entry2 = test2.get("doesn't", "exist");
//expectType<undefined>(test2Entry2)

const test2Entry3 = test2.has("doesn't", "exist");
expectType<boolean>(test2Entry3)

const test10 = constMap(sample, [[0, 1], [0, 1, 2]]);
const test10Entry1 = test10("Testnet", "Sepolia");
expectType<["Testnet", "Sepolia", 11155111n]>(test10Entry1)

const test20 = constMap(sample, [0, 1]);
const test20Entry1 = test20("Testnet");
expectType<["Ethereum", "Sepolia"]>(test20Entry1)

const test30 = constMap(sample, [2, 0]);
const test30Entry1 = test30(1n);
expectType<"Mainnet">(test30Entry1)

const test31 = constMap(sample, [2, [0, 1]]);
const test31Entry1 = test31(1n);
expectType<["Mainnet", "Ethereum"]>(test31Entry1)

const test31Entry2 = test31(11155111n);
expectType<["Testnet", "Sepolia"]>(test31Entry2)

const test40 = constMap(sample, [1, 0]);
const test40Entry1 = test40("Ethereum");
expectType<["Mainnet", "Testnet"]>(test40Entry1)
const test40Entry2 = test40("Sepolia");
expectType<["Testnet"]>(test40Entry2)
const test40Entry3 = test40("Bsc");
expectType<["Mainnet"]>(test40Entry3)
