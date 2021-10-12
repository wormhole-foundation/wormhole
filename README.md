# Pricecaster Service

This service consumes prices from "price fetchers" and feeds blockchain publishers. In case of Algorand publisher class, a TEAL program with messages containing signed price data. The program code validates signature and message validity, and if successful, subsequently stores the price information in the global application information for other contracts to retrieve.

All gathered price information is stored in a buffer by the Fetcher component -with a maximum size determined by settings-.  The price to get from that buffer is selected by the **IStrategy** class implementation; the default implementation being to get the most recent price and clear the buffer for new items to arrive. 

Alternative strategies for different purposes, such as getting averages and forecasting, can be implemented easily.

## System Overview

The Pricecaster backend can be configured with any class implementing **IPriceFetcher** and **IPublisher** interfaces. The following diagram shows the service operating with a fetcher from ![Pyth Network](https://pyth.network/), feeding the Algorand chain through the `StdAlgoPublisher` class.

![PRICECASTER](https://user-images.githubusercontent.com/4740613/136037362-bed34a49-6b83-42e1-821d-1df3d9a41477.png)

## Backend Configuration

The backend will read configuration from a `settings.ts` file pointed by the `PRICECASTER_SETTINGS` environment variable.

## Tests

At this time, there is a TEAL contract test that can be run with 

`npm run test`

Backend tests will come shortly.



