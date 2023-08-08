import { describe, expect, test } from "@jest/globals";
import {
  CircleIntegrationDeposit,
  CircleIntegrationPayload,
  parseCircleIntegrationDepositWithPayload,
} from "../circleIntegration";

describe("VAA Parsing Unit Tests", () => {
  test("CircleIntegration DepositWithPayload", () => {
    const testPayload = Buffer.from(
      "01000000000000000000000000b97ef9ef8734c71904d8002f8b6bc66dd9c48a6e0000000000000000000000000000000000000000000000000000000018701a8000000001000000000000000000000f880000000000000000000000004cb69fae7e7af841e44e1a1c30af640739378bb20000000000000000000000004cb69fae7e7af841e44e1a1c30af640739378bb20061010000000000000000000000000000000000000000000000000000000001c9c3800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000380d25e69c3df71de6891bfd511af87e9b3e9036",
      "hex"
    );
    const testResult: CircleIntegrationDeposit = {
      payloadType: CircleIntegrationPayload.DepositWithPayload,
      tokenAddress: Buffer.from(
        "000000000000000000000000b97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
        "hex"
      ),
      amount: BigInt("410000000"),
      sourceDomain: 1,
      targetDomain: 0,
      nonce: BigInt("3976"),
      fromAddress: Buffer.from(
        "0000000000000000000000004cb69fae7e7af841e44e1a1c30af640739378bb2",
        "hex"
      ),
      mintRecipient: Buffer.from(
        "0000000000000000000000004cb69fae7e7af841e44e1a1c30af640739378bb2",
        "hex"
      ),
      payloadLen: 97,
      depositPayload: Buffer.from(
        "010000000000000000000000000000000000000000000000000000000001c9c3800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000380d25e69c3df71de6891bfd511af87e9b3e9036",
        "hex"
      ),
    };
    const parsedPayload = parseCircleIntegrationDepositWithPayload(testPayload);
    expect(parsedPayload).toEqual(testResult);
  });
});
