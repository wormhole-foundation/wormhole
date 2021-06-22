/**
 * Checks if window is available. Helps determine wether the code being executed
 * is in a node context or a browser context.
 *
 * @export
 * @returns
 */
export default function isBrowser() {
  return typeof window !== 'undefined';
}
