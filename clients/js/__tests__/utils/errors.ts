export const YARGS_COMMAND_FAILED = "Command failed";

export const CONTRACT_NOT_DEPLOYED = (
  chain: string,
  type: "Core" | "NFTBridge" | "TokenBridge"
) => `${type} not deployed on ${chain}`;
