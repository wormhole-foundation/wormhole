# Memory allocation
| container          | measured VmHWM | set limit | service/job           |
|--------------------|----------------|-----------|-----------------------|
| guardiand          | 2777Mi         | 3300Mi    | service               |
| ganache            | 471Mi          | 500Mi     | service               |
| eth-deploy         | ?              | 1Gi       | job                   |
| mine               | 271Mi          | 300Mi     | service               |
| spy                | 116Mi          | 150Mi     | service               |
| algorand-postgres  | ?              | 80Mi      | service               |
| algorand-algod     | 644Mi          | 1000Mi    | service               |
| algorand-indexer   | ?              | 100Mi     | service               |
| algorand-contracts | ?              | 200Mi     | service, could be job |
| aptos-node         | 143Mi          | 500Mi     | service               |
| aptos-contracts    | ?              | 300Mi     | service, could be job |
| btc-node           | 310Mi          | 350Mi     | service               |
| near-node          | 639Mi          | 700Mi     | service               |
| near-deploy        | 462Mi          | 500Mi     | job                   |
| solana-devnet      | 1769Mi         | 2000Mi    | service               |
| solana-setup       | ?              | 750Mi     | service, could be job |
| spy-listener       | ?              | 150Mi     | service               |
| spy-relayer        | 76Mi           | 200Mi     | service               |
| spy-wallet-monitor | 81Mi           | 100Mi     | service               |
| spy                | 102Mi          | 120Mi     | service               |
| terra-terrad       | 343Mi          | 400Mi     | service               |
| terra-contracts    | ?              | 200Mi     | could be job          |
| fcd-postgres       | ?              | 50Mi      | service               |
| fcd-collector      | ?              | 500Mi     | service               |
| fcd-api            | ?              | 200Mi     | service               |
| wormchaind         | 559Mi          | 1000Mi    | service               |
| sdk-ci-tests       | ?              | 2500Mi    | job                   |
| spydk-ci-tests     | ?              | 1000Mi    | job                   |

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
    * e.g. for `algod`: `name=[a]lgod;P=$(ps aux | grep $name | awk '{print $2}'); grep ^VmHWM /proc/$P/status | awk '{print $2}'`