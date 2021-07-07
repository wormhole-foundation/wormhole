## Initializing BigTable instance

Once you've created a BigTable instance and a Service Account key, these Go scripts can create the table and column families to save event data.

Pass your BigTable connection info via args:
- the Google Cloud projectID
- BigTable instance name
- the path to a GCP Service Account with appropriate permissions

Invoke the script with the DB config options and `-setupDB` to create the table and column families, if they do not already exist. If they do already exists when the script runs, it will do nothing.

```bash
go run . \
  -project your-GCP-projectID \
  -instance your-BigTable-instance-name \
  -keyFilePath ./service-account-key.json \
  -setupDB
```

## Querying BigTable

Lookup a row by the `EmitterChain:TxID`:

```bash
go run . \
  -project wormhole-315720 \
  -instance wormhole-dev \
  -keyFilePath ./bigtable-admin.json \
  -queryRowKey ethereum:0xf15fe0e6dedef169a25696c577bfddf3d35707210ac641108646d7cfe507f174
```

Lookup all rows with a timestamp >= `queryPreviousMinutes` ago:

```bash
go run . \
  -project wormhole-315720 \
  -instance wormhole-dev \
  -keyFilePath ./bigtable-admin.json \
  -queryPreviousMinutes 120
```
