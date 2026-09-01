import { queryRegistrationsEvm } from "../../evm";
import { handler } from "./registrations";

jest.mock("../../evm", () => ({
  queryRegistrationsEvm: jest.fn(),
}));

const mockQuery = queryRegistrationsEvm as jest.Mock;

const emitter = (byte: string) => "0x" + byte.repeat(32);

describe("worm info registrations", () => {
  let logSpy: jest.SpyInstance;

  beforeEach(() => {
    logSpy = jest.spyOn(console, "log").mockImplementation(() => {});
    mockQuery.mockReset();
  });

  afterEach(() => {
    logSpy.mockRestore();
  });

  it("passes through registrations for chains the SDK dropped", async () => {
    // on-chain state predating the SDK cleanup: Blast (36) and Terra (3)
    // are still registered with the token bridge
    const onChain = {
      Bsc: emitter("11"),
      Blast: emitter("ab"),
      Terra: emitter("cd"),
    };
    mockQuery.mockResolvedValue(onChain);

    await handler({
      network: "mainnet",
      chain: "ethereum",
      module: "TokenBridge",
      verify: false,
    } as any);

    expect(mockQuery).toHaveBeenCalledWith(
      "Mainnet",
      "Ethereum",
      "TokenBridge"
    );
    expect(logSpy).toHaveBeenCalledWith(onChain);
  });

  it("verify mode tolerates deprecated-chain registrations in the results", async () => {
    mockQuery.mockResolvedValue({ Blast: emitter("ab") });

    await handler({
      network: "mainnet",
      chain: "ethereum",
      module: "TokenBridge",
      verify: true,
    } as any);

    const output = logSpy.mock.calls.map((call) => String(call[0])).join("\n");
    expect(output).toMatch(/succeeded|Mismatches/);
  });

  it("rejects querying a deprecated chain directly", async () => {
    await expect(
      handler({
        network: "mainnet",
        chain: "blast",
        module: "TokenBridge",
        verify: false,
      } as any)
    ).rejects.toThrow(
      "Blast was dropped from the SDK and has no live bridge; its registrations cannot be queried"
    );
    expect(mockQuery).not.toHaveBeenCalled();
  });
});
