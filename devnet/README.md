# Memory allocation
| container          | measured VmHWM | set limit | service/job           |
|--------------------|----------------|-----------|-----------------------|
| guardiand***       | 2777Mi         | 5000Mi    | service               |
| ganache            | 471Mi          | 500Mi     | service               |
| eth-deploy         | ?              | 1Gi       | service, could be job |
| eth-mine           | 271Mi          | 300Mi     | service               |
| spy**              | 116Mi          | 300Mi     | service               |
| algorand-postgres  | ?              | 80Mi      | service               |
| algorand-algod     | 644Mi          | 1000Mi    | service               |
| algorand-indexer   | 67Mi           | 500Mi     | service               |
| algorand-contracts | ?              | 300Mi     | service, could be job |
| aptos-node         | 1201Mi         | 2Gi       | service               |
| aptos-contracts    | ?              | 300Mi     | service, could be job |
| btc-node           | 310Mi          | 350Mi     | service               |
| near-node          | 639Mi          | 1000Mi    | service               |
| near-deploy        | 462Mi          | 500Mi     | service, could be job |
| solana-devnet      | 1769Mi         | 3000Mi    | service               |
| solana-setup       | ?              | 1Gi       | service, could be job |
| spy-listener       | ?              | 150Mi     | service               |
| spy-relayer        | 76Mi           | 200Mi     | service               |
| spy-wallet-monitor | 81Mi           | 100Mi     | service               |
| spy                | 102Mi          | 300Mi     | service               |
| terra-terrad       | 343Mi          | 2Gi       | service               |
| terra-contracts    | ?              | 200Mi     | could be job          |
| terra2-terrad      | 343Mi          | 2Gi       | service               |
| terra2-deploy      | ?              | 200Mi     | could be job          |
| fcd-postgres       | ?              | 50Mi      | service               |
| fcd-collector      | ?              | 500Mi     | service               |
| fcd-api            | ?              | 200Mi     | service               |
| sdk-ci-tests*      | ?              | 10Gi      | job                   |
| spydk-ci-tests     | ?              | 1000Mi    | job                   |

* when tests are run sequentially instead of in parallel, only 2500Mi are required.
** without race condition detector, 150Mi would be sufficient
*** with sequential tests and without race condition detector, 3300Mi would be sufficient

## Debugging
* Detecting oomkill:
    * The symptom of an oomkill is the container being killed.
    * oomkill messages should show up in `/var/log/messages`. E.g.: `Nov 27 01:26:57 hostname kernel: oom-kill:constraint=CONSTRAINT_MEMCG,nodemask=(null),cpuset=2d90ae0dc04950f146d262fd80f1da3b4bbc69cf059b2bb7c66616d35e4d3ffd,mems_allowed=0,oom_memcg=/docker/8b5e1b8686b3a5082ea70faa5809e3053bb1f75077ce19236582b6d67190a48a/kubepods/burstable/podda04e81e-3992-4dd3-aea5-112148b37b9e/2d90ae0dc04950f146d262fd80f1da3b4bbc69cf059b2bb7c66616d35e4d3ffd,task_memcg=/docker/8b5e1b8686b3a5082ea70faa5809e3053bb1f75077ce19236582b6d67190a48a/kubepods/burstable/podda04e81e-3992-4dd3-aea5-112148b37b9e/2d90ae0dc04950f146d262fd80f1da3b4bbc69cf059b2bb7c66616d35e4d3ffd,task=near-sandbox,pid=3723608,uid=0`
    * To get the exit status of a container, you can do `kubectl get pod -o jsonpath='{.items[*].status.containerStatuses[*].lastState.terminated}' | jq` and look at `exitCode`, which would be `137` and `reason`, which would be `OOMKilled`.
* kubectl top:
    * Enable metrics server: `minikube addons enable metrics-server`
    * Wait for ~60s (metrics are usually collected every minute)
    * `kubectl top pod --containers=true`
* Get the max memory consumption of a process over its lifetime:
    * e.g. for `algod`: `name="[a]lgod";P=$(ps aux | grep "$name" | awk '{print $2}'); grep ^VmHWM /proc/$P/status | awk '{print $2}'`