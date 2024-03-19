# Guidelines / Skeleton for guardian watcher

All of the watchers in the guardian should be written similarly so that if one understands the structure and function of one watcher, then that knowledge is directly applicable to the other watchers. If a bug is found in one watcher, it should be fairly simple to see if that bug exists in the other watchers. To that end this document intends to give guidelines for a watcher written in go.

### Responsibilities of a watcher:

1. Query the chain for the current block height
2. Receive messages from the chain’s wormhole core contract and emit them as observations in the common.MessagePublication format
3. Handle re-observation requests

### Watcher data structure:

The data for the watcher should be contained in a watcher struct. The following is an example of the minimal data that should be in the Watcher struct.

```go
Watcher struct {
  // The following should contain whatever parameters is needed to listen
  // to events from the core contract (like RPC, WS, Account, package, etc.).
  chainRPC  string

  // The following is the channel for emitting observations
  msgChan   chan *common.MessagePublication

  // The following is the channel for receiving re-observation requests
  obsvReqC  chan *gossipv1.ObservationRequest

  // Used to report the health of the watcher
  readiness.Component
}
```

### NewWatcher function

This function is used to instantiate a watcher and typically looks like this:

```go
func NewWatcher(
  chainID vaa.ChainID, // May be hard coded instead of passed in.
	chainRPC string,
	msgChan chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		chainRPC:      chainRPC,
		msgChan:       msgChan,
		obsvReqC:      obsvReqC,
		readinessSync: common.MustConvertChainIdToReadinessSyncing(chainID),
	}
}
```

### Prometheus:

The watcher should peg counts to be picked up by prometheus. The following are recommended:

```go
var (
  <chain>ConnectionErrors = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "wormhole_<chain>_connection_errors_total",
    Help: "Total number of <chain> connection errors",
  }, []string{"reason"})
  <chain>MessagesConfirmed = promauto.NewCounter(prometheus.CounterOpts{
    Name: "wormhole_<chain>_observations_confirmed_total",
    Help: "Total number of verified <chain> observations found",
  })
  current<chain>Height = promauto.NewGauge(prometheus.GaugeOpts{
    Name: "wormhole_<chain>_current_height",
    Help: "Current <chain> block height",
  })
)
```

### Functions:

The Run function is what is called by the guardian to start this watcher. It should do the following:

1. Create a connection to the blockchain with an appropriate subscription to listen to events from the core contract.
2. Signal readiness (This indicates this watcher is fully initialized and ready to work)
3. Create a go routine to get and process events from 1.
4. Create a go routine to periodically get the block height.
5. Create a go routine to watch for and handle re-observation requests.

The following is an outline in go. Notice that each go routine is encapsulated by RunWithScissors(). What this function does is wrap the go routine to make sure that any panic or error gets reported to an error channel. Other go routines use this as a mechanism to determine degraded performance of the watcher and exit.

<!-- cspell:disable -->
```go
func (e *Watcher) Run(ctx context.Context) error {
  // Setup a logger
  logger := supervisor.Logger(ctx)

  // Create a connection to the blockchain and subscribe to
  // core contract events here

  // Create the timer for the get_block_height go routine
  timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

  // Create an error channel
  errC := make(chan error)
	defer close(errC)

  // Signal that basic initialization is complete
  readiness.SetReady(e.readinessSync)

  // Signal to the supervisor that this runnable has finished initialization
  supervisor.Signal(ctx, supervisor.SignalHealthy)

  // Create the go routine to handle events from core contract
  common.RunWithScissors(ctx, errC, "core_events", func(ctx context.Context) error {
		logger.Error("Entering core_events...")
		for {
			select {
      case err := <-errC:
				logger.Error("core_events died", zap.Error(err))
				return fmt.Errorf("core_events died: %w", err)
			case <-ctx.Done():
				logger.Error("coreEvents context done")
				return ctx.Err()

      default:
        // Read events and handle them here
        // If this is a blocking read, then set readiness in the
        // get_block_height thread. Else, uncomment the following line:
        // readiness.SetReady()
      } // end select
    } // end for
  } // end RunWithScissors

  // Create the go routine to periodically get the block height
  common.RunWithScissors(ctx, errC, "get_block_height", func(ctx context.Context) error {
		for {
			select {
      case err := <-errC:
				logger.Error("get_block_height died", zap.Error(err))
				return fmt.Errorf("get_block_height died: %w", err)
			case <-ctx.Done():
				logger.Error("get_block_height context done")
				return ctx.Err()

			case <-timer.C:
        // Get the block height

        // Try to handle readiness in core_events go routine.
        // If core_events read is a blocking read, then handle
        // readiness here and uncomment the following line:
        // readiness.SetReady(e.readinessSync)
      } // end select
		} // end for
	}) // end RunWithScissors

  // Create the go routine to listen for re-observation requests
  common.RunWithScissors(ctx, errC, "fetch_obvs_req", func(ctx context.Context) error {
		for {
			select {
			case err := <-errC:
				logger.Error("fetch_obvs_req died", zap.Error(err))
				return fmt.Errorf("fetch_obvs_req died: %w", err)
			case <-ctx.Done():
				logger.Error("fetch_obvs_req context done")
				return ctx.Err()
			case r := <-e.obsvReqC:
				if vaa.ChainID(r.ChainId) != vaa.ChainID<this_chain> {
					panic("invalid chain ID")
				}
        // Handle the re-observation request
      } // end select
    } // end for
  }) // end RunWithScissors

  // This is done at the end of the Run function to cleanup as needed
  // and return the reason for Run() returning.
  select {
	case <-ctx.Done():
		// Close socket(s), if necessary
		return ctx.Err()
	case err := <-errC:
		// Close socket(s), if necessary
		return err
	} // end select
} // end Run()
```
<!-- cspell:enable -->

### Other thoughts / directions:

1. Which websocket package to use? (gorilla or nhooyr). nhooyr was selected for the following reasons:
   1. It has an active maintainer. Gorilla has been archived.
   2. It supports concurrent writes. Gorilla does not support concurrent writes
   3. It supports passing in a Context to read() for timing out the read.
2. The core_events and get_block_height go routines should be combined into a single go routine when the core_events reader supports reads with timeouts.
