# Wormhole

Read our [introduction blog article](https://medium.com/certus-one/introducing-the-wormhole-bridge-24911b7335f7) 
for more details on Wormhole and its major design decisions.

See [DEVELOP.md](DEVELOP.md) for instructions on how to set up a local devnet.

See [docs/operations.md](docs/operations.md) for node operator instructions.

![](docs/images/overview.svg)

### Audit / Feature Status

| Feature           | Maintainer | Auditor  | Status          |
|-------------------|------------|----------|-----------------|
| Ethereum contract | Certus One | Kudelski | ✔️ Audited      |
| Solana contract   | Certus One | Kudelski | ✔️ Audited      |
| Bridge node       | Certus One | Kudelski | 🚧 In Progress  |
| Terra contract    | Everstake  | Kudelski | ⚠ Pending       |

⚠ **This software is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing permissions and limitations under the License.** Or plainly
spoken - this is a very complex software which targets a bleeding-edge, experimental smart contract runtime. Mistakes
happens, and no matter how hard you try and whether or not you pay someone to audit it, it may eat your tokens, set your
printer on fire or startle your cat.

Cryptocurrencies in general are a high-risk investment, there's decent chance you'll lose your money, and you most
certainly shouldn't put your life savings into a Wormhole contract (or any other).

### Repo overview

- **[bridge/](bridge/)** — The guardian node which connects to both chains, observes lockups and submits VAAs.
  Written in pure Go.
  
  - [cmd/](bridge/cmd/) - CLI entry point, deals with the mechanics of parsing command line flags and loading keys.
  - **[pkg/processor](bridge/pkg/processor)** — Most of the business logic for cross-chain communication
    lives here. Talks to multiple loosely coupled services communicating via Go channels.
  - [pkg/p2p](bridge/pkg/p2p) — libp2p-based gossip network.
  - [pkg/devnet](bridge/pkg/devnet) — Constants and helper functions for the deterministic local devnet.
  - [pkg/ethereum](bridge/pkg/ethereum) — Ethereum chain interface with auto-generated contract ABI.
    Uses go-ethereum to directly connect to an Eth node.
  - [pkg/solana](bridge/pkg/ethereum) — Solana chain interface. Light gRPC wrapper around a Rust agent (see below)
    which actually talks to Solana.  
  - [pkg/supervisor](bridge/pkg/supervisor) — Erlang-inspired process supervision tree imported from Certus One's
    internal code base. We use this everywhere in the bridge code for fault tolerance and fast convergence.
  - [pkg/vaa](bridge/pkg/vaa) — Go implementation of our VAA structure, including serialization code.
  
- **[ethereum/](ethereum/)** — Ethereum wormhole contract, tests and fixtures.

  - **[contracts/](ethereum/contracts)** — Wormhole itself, the wrapped token and helper libraries.
  - [migrations/](ethereum/migrations) — Ganache migration that deploys the contracts to a local devnet.
    This is the starting point for both the tests and the devnet. Note that devnet and tests result
    in different devnet states.
  - [src/send-lockups.js](ethereum/src/send-lockups.js) — Sends ETH lockups in a loop.
    See DEVELOP.md for usage.
  
- **[solana/](solana/)** — Solana sidecar agent, contract and CLI.
  - **[agent/](solana/agent/)** — Rust agent sidecar deployed alongside each Guardian node. It serves
    a local gRPC API to interface with the Solana blockchain. This is far easier to maintain than a
    pure-Go Solana client.
  - **[bridge/](solana/bridge/)** — Solana Wormhole smart contract code. 
  - [cli/](solana/cli/) — Wormhole user CLI tool for interaction with the smart contract. 
  - [devnet_setup.sh](solana/devnet_setup.sh) — Devnet initialization and lockup generator
    (the Solana equivalent to the Ganache migration + send-lockups.js). Runs as a sidecar alongside the Solana devnet. 

- **[proto/](proto/)** — Protocol Buffer definitions for the P2P network and the local Solana agent RPC.
  These are heavily commented and a good intro.

- **[third_party/](third_party/)** — Build machinery and tooling for third party applications we use.
  - [abigen/](third_party/abigen/) — Reproducible build for the go-ethereum ABI code generator we use.
  - **[solana/](third_party/solana/)** — Build for the full Solana project plus a floating patchset we maintain while
    waiting for features to be implemented in the upstream project. 

- **[docs/](docs/)** — Operator documentation and project specs.

- **[web/](web/)** — User interface for cross-chain transfers. Not yet wired into the local devnet.
  Uses Metamask and Web3.js to initiate transfers from a browser.
  Watch [this video](https://youtu.be/9OTTyJ_h4O0) as an introduction.
  
- [tools/](tools/) — Reproducible builds for local development tooling like buf and protoc-gen-go. 
  
- [Tiltfile](Tiltfile),  [devnet/](devnet/) and various Dockerfiles — deployment code and fixtures for local development.
  Deploys a deterministic devnet with an Ethereum devnet, Solana devnet, and a variably sized guardian set
  that can be used to simulate full cross-chain transfers. The Dockerfiles are carefully designed for fast incremental
  builds with local caching, and require a recent Docker version with Buildkit support. See DEVELOP.md for usage.
  
- [generate-abi.sh](generate-abi.sh) and [generate-protos.sh](generate-protos.sh) — 
  Helper scripts to (re-)build generated code. The Eth ABI is committed to the repo, so you only
  need to run this script if the Wormhole.sol interface changes. The protobuf libraries are not
  committed and will be regenerated automatically by the Tiltfile. 
  
