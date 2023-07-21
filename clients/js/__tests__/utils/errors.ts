export const YARGS_COMMAND_FAILED = "Command failed";

export const CONTRACT_NOT_DEPLOYED = (
  chain: string,
  type: "Core" | "NFTBridge" | "TokenBridge"
) => `${type} not deployed on ${chain}`;

export const INVALID_VAA_CHAIN = (expectedChain: string, vaaChain: string) => {
  return `Error: Specified target chain (${expectedChain}) does not match VAA target chain (${vaaChain})`;
};
