export function errorMsg(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function errorStack(error: unknown): string {
  return String(error instanceof Error ? error.stack : error);
}