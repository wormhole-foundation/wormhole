This tool can be used to generate the list of tokens to be monitored by the chain governor.
It works by querying the notional TVL data from Portal and populating the tokens.go file in
the governor package with everything over the configured minimal notional value.

To run this tool, do:

```
npm ci
npm run start
```

You can then commit the updated version of node/pkg/governor/tokens.go.
