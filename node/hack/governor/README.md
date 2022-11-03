# Overview

This tool can be used to generate the list of tokens to be monitored by the chain governor.
It works by querying the notional TVL data from Portal and populating the generated_tokens.go file in
the governor package with everything over the hard coded minimal notional value.

## Configuration
To update the minimal notional value, edit src/index.ts and change the value of MinNotional.

## Always Included Tokens
Additionally, you can create an include_list.csv file in this directory where the contents are
of the form "<originChain>,<nativeTokenAddress>", and all tokens listed there will be included
in the generated token list, regardless of their notional value.

## Running the script
To run this tool, do:

```
npm ci
npm run start
```

## Manually Included Tokens
The governor also makes use of a list of manually added tokens. These are tokens that do not exist
in the notional TVL data. These tokens are listed in wormhole/node/pkg/governor/manual_tokens.go

## Verifying the Token Lists
To verify that the Coin Gecko query still works with the new token list, do:
```
go run check_query.go
```

Before committing the generated file, you should run the governor tests and ensure that they pass:
```
cd wormhole/node/pkg/governor
go test
```

## Committing the Changes
You can then commit the updated version of node/pkg/governor/generated_tokens.go.
