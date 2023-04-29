import { MoveAbort } from "./moveAbort";

export function parseWormholeError(errorMessage: string) {
  const abort = MoveAbort.parseError(errorMessage);
  const code = abort.errorCode;

  switch (abort.moduleName) {
    case "required_version": {
      switch (code) {
        case 0n: {
          return "E_OUTDATED_VERSION";
        }
        default: {
          throw new Error(`unrecognized error code: ${abort}`);
        }
      }
    }
    default: {
      throw new Error(`unrecognized module: ${abort}`);
    }
  }
}
