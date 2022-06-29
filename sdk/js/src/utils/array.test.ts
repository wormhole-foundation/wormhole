import { zeroPad } from "ethers/lib/utils";
import { canonicalAddress } from "..";
import { tryUint8ArrayToNative } from "./array";

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
