/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceFetcher } from './pricefetch'

const pricefetcher = new PriceFetcher()
pricefetcher.run()
