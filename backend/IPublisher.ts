/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceTicker } from './PriceTicker'

export class PublishInfo {
    block: BigInt = BigInt(0)
    txid: string = ''
}

export interface IPublisher {
    publish(tick: PriceTicker): Promise<PublishInfo>
}
