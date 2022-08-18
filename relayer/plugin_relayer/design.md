Relayer process outline:

1. Subscribe & listen to certain Core Contract events via REST and/or via SPY. (Prior to event)
2. Process event and decide if action is needed
3. Queue Action
4. process action

Interfaces:

Filtering:

- input: none
- output: Emitter chain & addresses to filter for

Listener:

- input: VAA & Immutable action state (redis tables)
- output: 
  - New actions to be enqueued for exucution
  - New actions to be side-lined

Execute:

- input: wallet toolbox, immutable action state, action
- output: New actions to enqueue, actions to dequeue

## Redis tables
Each plugin will get its own redis sandbox, which is isolated from other plugins running on the same relayer.
Each plugin receives two of the sandboxed tables:
  - Primary list of pending actions
  - Listener 'staging' area

## Workers
- 1 sync worker per wallet, with actions created by multiple plugins 
  - 1 signer wallet
  - all providers 
  - redis state for plugin that owns this action
- N async read-only actions scheduled in parallel  
  - all providers
  - redis state for plugin that owns this action
- (later) actions can also be async, relayer will schedule multiple async actions together

## Config 
### Base Relayer
- common
- listener
- executor 

### Plugin 
- defines config type
- bundles config for mainnet and devnet as default
- plugin constructor gets passed:
  - which env (to look up in default map)
  - optionally override configs
  - common config (cannot know whether it's running in executor or listener process)