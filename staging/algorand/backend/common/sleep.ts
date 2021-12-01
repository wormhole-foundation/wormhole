/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

export function sleep (ms: number) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}
