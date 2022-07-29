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
    "wormholechain"
  );
  expect(native).toBe(human);

  expect(tryNativeToHexString(human, "wormholechain")).toBe(canonical);
});
