import { expect, test } from "@jest/globals";
import { zeroPad } from "ethers/lib/utils";
import { canonicalAddress } from "../cosmos";
import { tryUint8ArrayToNative, tryNativeToHexString } from "./array";

test("terra address conversion", () => {
  const human = "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v";
  const canonical = canonicalAddress(human);
  const lpadCanonical = zeroPad(canonical, 32);
  const nativeClassic = tryUint8ArrayToNative(lpadCanonical, "terra");
  expect(nativeClassic).toBe(human);
  const native2 = tryUint8ArrayToNative(lpadCanonical, "terra2");
  expect(native2).toBe(human);
  // terra 2 contracts are 32 bytes
  const humanContract =
    "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au";
  const canonicalContract = canonicalAddress(humanContract);
  const nativeContract = tryUint8ArrayToNative(canonicalContract, "terra2");
  expect(nativeContract).toBe(nativeContract);
  // TODO: native to hex is wrong, which we should correct
});

test("wormchain address conversion", () => {
  const human = "wormhole1ap5vgur5zlgys8whugfegnn43emka567dtq0jl";
  const canonical =
    "000000000000000000000000e868c4707417d0481dd7e213944e758e776ed35e";
  const native = tryUint8ArrayToNative(
    new Uint8Array(Buffer.from(canonical, "hex")),
    "wormchain"
  );
  expect(native).toBe(human);

  expect(tryNativeToHexString(human, "wormchain")).toBe(canonical);
});

test("wormchain address conversion2", () => {
  const human =
    "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh";
  const canonical =
    "aeb534c45c3049d380b9d9b966f9895f53abd4301bfaff407fa09dea8ae7a924";
  const native = tryUint8ArrayToNative(
    new Uint8Array(Buffer.from(canonical, "hex")),
    "wormchain"
  );
  expect(native).toBe(human);

  expect(tryNativeToHexString(human, "wormchain")).toBe(canonical);
});

test("wormchain address conversion no leading 0s", () => {
  const human = "wormhole1yre8d0ek4vp0wjlec407525zjctq7t32z930fp";
  const canonical = "20f276bf36ab02f74bf9c55fea2a8296160f2e2a";
  const native = tryUint8ArrayToNative(
    new Uint8Array(Buffer.from(canonical, "hex")),
    "wormchain"
  );
  expect(native).toBe(human);

  // Can't do the reverse because the supplied canonical does not have leading 0s
  // expect(tryNativeToHexString(human, "wormchain")).toBe(canonical);
});

test("injective address conversion", () => {
  const human = "inj180rl9ezc4389t72pc3vvlkxxs5d9jx60w9eeu3";
  const canonical = canonicalAddress(human);
  const lpadCanonical = zeroPad(canonical, 32);
  const native = tryUint8ArrayToNative(lpadCanonical, "injective");
  expect(native).toBe(human);
});

test("sei address conversion", () => {
  const human =
    "sei189adguawugk3e55zn63z8r9ll29xrjwca636ra7v7gxuzn98sxyqwzt47l";
  const canonical =
    "397ad473aee22d1cd2829ea2238cbffa8a61c9d8eea3a1f7ccf20dc14ca78188";
  const native = tryUint8ArrayToNative(
    new Uint8Array(Buffer.from(canonical, "hex")),
    "sei"
  );
  expect(native).toBe(human);
});
