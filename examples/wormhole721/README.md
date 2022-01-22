# Wormhole721 & Wormhole721Upgradable

Implementation of ERC721 and ERC721Upgradable NFTs using Wormhole to be natively cross-chain.

## Usage

Install:
```
npm install @ndujalabs/wormhole721
```

NFT contact:
```js
import "@openzeppelin/contracts-upgradeable/token/ERC721/ERC721Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721EnumerableUpgradeable.sol";
import "@ndujalabs/wormhole721/contracts/Wormhole721Upgradeable.sol";
...

contract MyBeautifulNFT is
...
ERC721Upgradeable,
ERC721EnumerableUpgradeable,
Wormhole721Upgradeable
{
  ...
  function initialize(uint256 lastTokenId_, bool secondaryChain) public initializer {
    __Wormhole721_init("My Beautiful NFT", "MBNFT");
    __ERC721Enumerable_init();
    ...
  }
  ...
}
```

## Examples

For a real-world example, see the [Everdragons2 Genesis contact](https://github.com/ndujaLabs/everdragons2-core).
