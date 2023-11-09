import { test, describe, expect } from "@jest/globals";
import { ConsistencyLevels, finalityThreshold, consistencyLevelToBlock } from "../src/constants/finality";

describe("Finality tests", function () {
  test("Receive expected number of rounds", () => {
    expect(finalityThreshold("Ethereum")).toEqual(64);
    expect(finalityThreshold("Algorand")).toEqual(0);
    expect(finalityThreshold("Solana")).toEqual(32);
  })

  //test.each([chains])("Accesses Finalization thresholds", (chain) => {
  //  expect(finalityThreshold(chain)).toBeTruthy();
  //})

  const fromBlock = 100n;
  test("Estimates rounds from instant consistency level", () => {
    expect(consistencyLevelToBlock("Algorand", ConsistencyLevels.Immediate, fromBlock)).toEqual(fromBlock);
    expect(consistencyLevelToBlock("Solana", ConsistencyLevels.Immediate, fromBlock)).toEqual(fromBlock);
    expect(consistencyLevelToBlock("Terra", ConsistencyLevels.Immediate, fromBlock)).toEqual(fromBlock);
  })

  test("Estimates rounds from safe consistency level", () => {
    // 100 + (32 - (100 % 32))
    expect(consistencyLevelToBlock("Ethereum", ConsistencyLevels.Safe, fromBlock)).toEqual(128n);
    // 100 + consistency level as rounds                
    expect(consistencyLevelToBlock("Bsc", ConsistencyLevels.Safe, fromBlock)).toEqual(301n);
    // 100 + 0 (instant)                                
    expect(consistencyLevelToBlock("Algorand", ConsistencyLevels.Safe, fromBlock)).toEqual(100n);
  })

  test("Estimates rounds from finalized consistency level", () => {
    // 100 + (# final rounds)
    expect(consistencyLevelToBlock("Ethereum", ConsistencyLevels.Finalized, fromBlock)).toEqual(fromBlock + 64n);
    expect(consistencyLevelToBlock("Solana", ConsistencyLevels.Finalized, fromBlock)).toEqual(fromBlock + 32n);
    // 100 + 0 (instant)
    expect(consistencyLevelToBlock("Algorand", ConsistencyLevels.Finalized, fromBlock)).toEqual(fromBlock);
    // L2 required but not factored into estimate
    expect(consistencyLevelToBlock("Polygon", ConsistencyLevels.Finalized, fromBlock)).toEqual(fromBlock + 512n);
  })

});

