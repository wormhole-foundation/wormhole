Relayer process outline:

1. Subscribe & listen to certain Core Contract events via REST and/or via SPY. (Prior to event)
2. Process event and decide if action is needed
3. Queue Action
4. process action

Interfaces:

Filtering:

- input: none
- output: Emitter chain & addresses to filter for

Schedule:

- input: VAA & Immutable action state
- output: New actions to be enqueued

Execute:

- input: wallet toolbox, immutable action state, action
- output: New actions to enqueue, actions to dequeue
