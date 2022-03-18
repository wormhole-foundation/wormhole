## Google Cloud functions for BigTable

This is a reference implementaion for getting data out of BigTable.

## Contents

This directory holds GCP Cloud Functions, one per file, along with shared utilities in `shared.go`. The file names correspond to the hosted endpoints. ie endpoint `.../*-notionaltvl` is the file `notional-tvl.go`

## Debugging with VSCode

### prereqs

- Golang >= 1.16 installed and available on your path.
- The Go VSCode extension, and gopls installed.

### IDE setup

- open a new VSCode window
- File menu --> "Open Workspace from File..."
- Select `event_database/cloud_functions/workspace.code-workspace`

Opening the workspace file as described above will open both `cloud_functions` and `functions_server`, so that you get all the VSCode goodness of intellesense, ability to run the code with the Go debugger, set breakpoints, etc.

Add your environment variables to `functions_server/.vscode/launch.json`

Start the debug server by pressing `F5`. You can check your server is up by requesting http://localhost:8080/readyz.

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
