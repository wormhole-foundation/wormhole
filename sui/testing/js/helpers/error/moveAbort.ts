export function parseMoveAbort(errorMessage: string) {
  const parsed = errorMessage.matchAll(
    /MoveAbort\(MoveLocation { module: ModuleId { address: ([0-9a-f]{64}), name: Identifier\("([A-Za-z_]+)"\) }, function: ([0-9]+), instruction: ([0-9]+), function_name: Some\("([A-Za-z_]+)"\) }, ([0-9]+)\) in command ([0-9]+)/g
  );

  return parsed.next().value.slice(1, 8);
}

export class MoveAbort {
  packageId: string;
  moduleName: string;
  functionName: string;
  errorCode: bigint;
  command: number;

  constructor(
    packageId: string,
    moduleName: string,
    functionName: string,
    errorCode: string,
    command: string
  ) {
    this.packageId = packageId;
    this.moduleName = moduleName;
    this.functionName = functionName;
    this.errorCode = BigInt(errorCode);
    this.command = Number(command);
  }

  static parseError(errorMessage: string): MoveAbort {
    const [packageId, moduleName, , , functionName, errorCode, command] =
      parseMoveAbort(errorMessage);

    return new MoveAbort(
      "0x" + packageId,
      moduleName,
      functionName,
      errorCode,
      command
    );
  }
}
