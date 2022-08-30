This tool can be used to generate the list of tokens to be monitored by the chain governor.
It works by querying the notional TVL data from Portal and populating the tokens.go file in
the governor package with everything over the hard coded minimal notional value.

To update the minimal notional value, edit src/index.ts and change the value of MinNotional.

Additionally, you can create an include_list.csv file in this directory where the contents are
of the form "<originChain>,<nativeTokenAddress>", and all tokens listed there will be included
in the generated token list, regardless of their notional value.

To run this tool, do:

```
npm ci
npm run start
```

To verify that the Coin Gecko query still works with the new token list, do:
```
go run check_query.go
```

You can then commit the updated version of node/pkg/governor/tokens.go.
