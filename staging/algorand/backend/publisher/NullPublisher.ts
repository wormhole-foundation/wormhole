import { IPublisher, PublishInfo } from '../publisher/IPublisher'
import { PriceTicker } from '../common/priceTicker'

export class NullPublisher implements IPublisher {
  start (): void {
    throw new Error('Method not implemented.')
  }

  stop (): void {
    throw new Error('Method not implemented.')
  }

  publish (tick: PriceTicker): Promise<PublishInfo> {
    throw new Error('Method not implemented.')
  }
}
