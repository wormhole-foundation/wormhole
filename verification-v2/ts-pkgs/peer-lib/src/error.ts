import { inspect } from "util";

export function errorMsg(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function errorStack(error: unknown): string {
  return inspect(error, {depth: 5});
}