## Google Cloud function for reading BigTable

This is a reference implementaion for getting data out of BigTable.

### deploying

First deploy (creation) must include all the flags to configure the environment:

    gcloud functions --project your-project deploy testnet --region europe-west3 --entry-point Entry --runtime go116 --trigger-http --allow-unauthenticated --service-account=your-readonly@your-project.iam.gserviceaccount.com --update-env-vars GCP_PROJECT=your-project,BIGTABLE_INSTANCE=wormhole-testnet

    gcloud functions --project your-project deploy processvaa-testnet --region europe-west3 --entry-point ProcessVAA --runtime go116 --trigger-topic new-vaa-testnet --service-account=your-readonly@your-project.iam.gserviceaccount.com --update-env-vars GCP_PROJECT=your-project,BIGTABLE_INSTANCE=wormhole-testnet

Subsequent deploys (updates) only need include flags to indentify the resource for updating: project, region, name.

    gcloud functions --project your-project deploy testnet --region europe-west3 --entry-point Entry

    gcloud functions --project your-project deploy processvaa-testnet --region europe-west3 --entry-point ProcessVAA

### invocation

All routes accept their input(s) as query parameters, or request body. Just two different ways of querying:

GET

```bash
curl "https://region-project-id.cloudfunctions.net/testnet/readrow?emitterChain=2&emitterAddress=000000000000000000000000e982e462b094850f12af94d21d470e21be9d0e9c&sequence=0000000000000006"
```

POST

```bash
curl -X POST  https://region-project-id.cloudfunctions.net/testnet/readrow \
-H "Content-Type:application/json" \
-d \
'{"emitterChain":"2", "emitterAddress":"000000000000000000000000e982e462b094850f12af94d21d470e21be9d0e9c", "sequence":"0000000000000006"}'

```

See [./bigtable-endpoints.md](./bigtable-endpoints.md) for API patterns
