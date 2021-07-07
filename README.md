# Wormhole v2

See [DEVELOP.md](DEVELOP.md) for instructions on how to set up a local devnet, and
[CONTRIBUTING.md](CONTRIBUTING.md) for instructions on how to contribute to this project.

See [docs/operations.md](docs/operations.md) for node operator instructions.

![](docs/images/overview.svg)

⚠ **Wormhole v2 is in active development - see "main" branch for the v1 mainnet version** ⚠

### Audit / Feature Status

⚠ **This software is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing permissions and limitations under the License.** Or plainly
spoken - this is a very complex piece of software which targets a bleeding-edge, experimental smart contract runtime.
Mistakes happen, and no matter how hard you try and whether you pay someone to audit it, it may eat your tokens, set
your printer on fire or startle your cat. Cryptocurrencies are a high-risk investment, no matter how fancy.

### READ FIRST BEFORE USING WORMHOLE

- Much of the Solana ecosystem uses wrapped assets issued by a centralized bridge operated by FTX (the "Sollet bridge").
  Markets on Serum or Raydium are using those centralized assets rather than Wormhole wrapped assets. These have names
  like "Wrapped BTC" or "Wrapped ETH". Wormhole is going to replace the FTX bridge eventually, but this will take some
  time - meanwhile, **Wormhole wrapped assets aren't terribly useful yet since there're no market for them.**
  
- Other tokens on Solana like USDC and USDT are **centralized native tokens issued on multiple chains**. If you transfer
  USDT from Ethereum to Solana, you will get "Wormhole Wrapped USDT" (wwUSDT), rather than native USDT.
  
- **Solana's SPL tokens have no on-chain metadata**. Wormhole can't know the name of the token when you
  transfer assets to Ethereum. All tokens are therefore named "WWT" plus the address of the SPL token.
  The reverse is also true - Wormhole knows the name of the ERC20 token, but there's no way to store it on Solana.
  There's an [off-chain name registry](https://github.com/solana-labs/token-list) that some block explorers use, but
  if you transfer an uncommon token to Solana, it may not show with a name on block explorers.
