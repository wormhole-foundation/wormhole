# Pricecaster Service

This service consumes prices from "price fetchers" and feeds blockchain publishers. In case of Algorand publisher class, a TEAL program with messages containing signed price data. The program code validates signature and message validity, and if successful, subsequently stores the price information in the global application information for other contracts to retrieve.

All gathered price information is stored in a buffer by the Fetcher component -with a maximum size determined by settings-.  The price to get from that buffer is selected by the **IStrategy** class implementation; the default implementation being to get the most recent price and clear the buffer for new items to arrive. 

Alternative strategies for different purposes, such as getting averages and forecasting, can be implemented easily.

## System Overview

The Pricecaster backend can be configured with any class implementing **IPriceFetcher** and **IPublisher** interfaces. The following diagram shows the service operating with a fetcher from ![Pyth Network](https://pyth.network/), feeding the Algorand chain through the `StdAlgoPublisher` class.

![PRICECASTER](https://user-images.githubusercontent.com/4740613/136037362-bed34a49-6b83-42e1-821d-1df3d9a41477.png)


## Data Format

### Input Message

The TEAL contract expects a fixed-length message consisting of:

```
  Field size
  9           header      Literal "PRICEDATA"
  1           version     int8 (Must be 1)
  8           dest        This appId 
  16          symbol      String padded with spaces e.g ("ALGO/USD        ")
  8           price       Price. 64bit integer.
  8           priceexp    Price exponent. Interpret as two-compliment, Big-Endian 64bit
  8           conf        Confidence (stdev). 64bit integer. 
  8           slot        Valid-slot of this aggregate price.
  8           ts          timestamp of this price submitted by PriceFetcher service
  32          s           Signature s-component
  32          r           Signature r-component 

  Size: 138 bytes. 
```

### Global state

The global state that is mantained by the contract consists of the following fields:

```
sym      : byte[] Symbol to keep price for   
vaddr    : byte[] Validator account          
price    : uint64 current price 
stdev    : uint64 current confidence (standard deviation)
slot     : uint64 slot of this onchain publication
exp      : byte[] exponent. Interpret as two-compliment, Big-Endian 64bit
ts       : uint64 last timestamp
```

#### Price parsing

The exponent is stored as a byte array containing a signed, two-complement 64-bit Big-Endian integer, as some networks like Pyth publish negative values here. For example, to parse the byte array from JS:

```
    const stExp = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'exp')
    const bufExp = Buffer.from(stExp, 'base64')
    const val = bufExp.readBigInt64BE()
```

## Backend Configuration

The backend will read configuration from a `settings.ts` file pointed by the `PRICECASTER_SETTINGS` environment variable.

## Tests

At this time, there is a TEAL contract test that can be run with 

`npm run test`

Backend tests will come shortly.



