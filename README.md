# Pricecaster Service

This service consumes prices from Pyth and feeds a TEAL program with messages containing signed price data. The program code validates signature and message validity, and if successful, subsequently stores the price information in the global application information for other contracts to retrieve.

## Backend Configuration

The fetcher will get information as soon as Pyth reports a price-change. Since publishing to the pricekeeper contract will be much slower, a buffered is approach is taken where a last set of prices is kept.

The number of prices kept is controlled by the `buffersize` setting.

The ratio of message publications currently is to publish again as soon as the last call finished and there is any buffer data available. This is configured with the `ratio` setting.

As prices may vary greatly in markets in short periods of time between publications, a set of strategies are provided to decide how to select data from the buffer.

Available strategies are:  
* `avg` Select the average price in-buffer.
* `wavg` Select the weighted-by-confidence average prices in buffer.
* `maxconf` Select the price with the maximum confidence (lowest deviation between publishers)

Enabling the 'phony' setting will **simulate** publications but no real calls will be made. Just useful for debugging.
