// contracts/TokenImplementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./TokenState.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Context.sol";
import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";

// Based on the OpenZepplin ERC20 implementation, licensed under MIT
contract TokenImplementation is TokenState, Context {
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    function initialize(
        string memory name_,
        string memory symbol_,
        uint8 decimals_,
        uint64 sequence_,
        address owner_,
        uint16 chainId_,
        bytes32 nativeContract_
    ) initializer public {
        _initializeNativeToken(
            name_,
            symbol_,
            decimals_,
            sequence_,
            owner_,
            chainId_,
            nativeContract_
        );

        // initialize w/ EIP712 state variables for domain separator
        _initializePermitStateIfNeeded();
    }

    function _initializeNativeToken(
        string memory name_,
        string memory symbol_,
        uint8 decimals_,
        uint64 sequence_,
        address owner_,
        uint16 chainId_,
        bytes32 nativeContract_
    ) internal {
        _state.name = name_;
        _state.symbol = symbol_;
        _state.decimals = decimals_;
        _state.metaLastUpdatedSequence = sequence_;

        _state.owner = owner_;

        _state.chainId = chainId_;
        _state.nativeContract = nativeContract_;
    }

    function _initializePermitStateIfNeeded() internal {
        // If someone were to change the implementation of name(), we
        // need to make sure we recache.
        bytes32 hashedName = _eip712DomainNameHashed();

        // If for some reason the salt generation changes with newer
        // token implementations, we need to make sure the state reflects
        // the new salt.
        bytes32 salt = _eip712DomainSalt();

        // check cached values
        if (_state.cachedHashedName != hashedName || _state.cachedSalt != salt) {
            _state.cachedChainId = block.chainid;
            _state.cachedThis = address(this);
            _state.cachedDomainSeparator = _buildDomainSeparator(hashedName, salt);
            _state.cachedSalt = salt;
            _state.cachedHashedName = hashedName;
        }
    }

    function name() public view returns (string memory) {
        return _state.name;
    }

    function symbol() public view returns (string memory) {
        return _state.symbol;
    }

    function owner() public view returns (address) {
        return _state.owner;
    }

    function decimals() public view returns (uint8) {
        return _state.decimals;
    }

    function totalSupply() public view returns (uint256) {
        return _state.totalSupply;
    }

    function chainId() public view returns (uint16) {
        return _state.chainId;
    }

    function nativeContract() public view returns (bytes32) {
        return _state.nativeContract;
    }

    function balanceOf(address account_) public view returns (uint256) {
        return _state.balances[account_];
    }

    function transfer(address recipient_, uint256 amount_) public returns (bool) {
        _transfer(_msgSender(), recipient_, amount_);
        return true;
    }

    function allowance(address owner_, address spender_) public view returns (uint256) {
        return _state.allowances[owner_][spender_];
    }

    function approve(address spender_, uint256 amount_) public returns (bool) {
        _approve(_msgSender(), spender_, amount_);
        return true;
    }

    function transferFrom(address sender_, address recipient_, uint256 amount_) public returns (bool) {
        _transfer(sender_, recipient_, amount_);

        uint256 currentAllowance = _state.allowances[sender_][_msgSender()];
        require(currentAllowance >= amount_, "ERC20: transfer amount exceeds allowance");
        _approve(sender_, _msgSender(), currentAllowance - amount_);

        return true;
    }

    function increaseAllowance(address spender_, uint256 addedValue_) public returns (bool) {
        _approve(_msgSender(), spender_, _state.allowances[_msgSender()][spender_] + addedValue_);
        return true;
    }

    function decreaseAllowance(address spender_, uint256 subtractedValue_) public returns (bool) {
        uint256 currentAllowance = _state.allowances[_msgSender()][spender_];
        require(currentAllowance >= subtractedValue_, "ERC20: decreased allowance below zero");
        _approve(_msgSender(), spender_, currentAllowance - subtractedValue_);

        return true;
    }

    function _transfer(address sender_, address recipient_, uint256 amount_) internal {
        require(sender_ != address(0), "ERC20: transfer from the zero address");
        require(recipient_ != address(0), "ERC20: transfer to the zero address");

        uint256 senderBalance = _state.balances[sender_];
        require(senderBalance >= amount_, "ERC20: transfer amount exceeds balance");
        _state.balances[sender_] = senderBalance - amount_;
        _state.balances[recipient_] += amount_;

        emit Transfer(sender_, recipient_, amount_);
    }

    function mint(address account_, uint256 amount_) public onlyOwner {
        _mint(account_, amount_);
    }

    function _mint(address account_, uint256 amount_) internal {
        require(account_ != address(0), "ERC20: mint to the zero address");

        _state.totalSupply += amount_;
        _state.balances[account_] += amount_;
        emit Transfer(address(0), account_, amount_);
    }

    function burn(address account_, uint256 amount_) public onlyOwner {
        _burn(account_, amount_);
    }

    function _burn(address account_, uint256 amount_) internal {
        require(account_ != address(0), "ERC20: burn from the zero address");

        uint256 accountBalance = _state.balances[account_];
        require(accountBalance >= amount_, "ERC20: burn amount exceeds balance");
        _state.balances[account_] = accountBalance - amount_;
        _state.totalSupply -= amount_;

        emit Transfer(account_, address(0), amount_);
    }

    function _approve(address owner_, address spender_, uint256 amount_) internal virtual {
        require(owner_ != address(0), "ERC20: approve from the zero address");
        require(spender_ != address(0), "ERC20: approve to the zero address");

        _state.allowances[owner_][spender_] = amount_;
        emit Approval(owner_, spender_, amount_);
    }

    function updateDetails(string memory name_, string memory symbol_, uint64 sequence_) public onlyOwner {
        require(_state.metaLastUpdatedSequence < sequence_, "current metadata is up to date");

        _state.name = name_;
        _state.symbol = symbol_;
        _state.metaLastUpdatedSequence = sequence_;

        // Because the name is updated, we need to recache the domain separator.
        // For old implementations, none of the caches may have been written to yet.
        _initializePermitStateIfNeeded();
    }

    modifier onlyOwner() {
        require(owner() == _msgSender(), "caller is not the owner");
        _;
    }

    modifier initializer() {
        require(
            !_state.initialized,
            "Already initialized"
        );

        _state.initialized = true;

        _;
    }

    /**
     * @dev Returns the domain separator for the current chain.
     */
    function _domainSeparatorV4() internal view returns (bytes32) {
        if (address(this) == _state.cachedThis && block.chainid == _state.cachedChainId) {
            return _state.cachedDomainSeparator;
        } else {
            return _buildDomainSeparator(
                _eip712DomainNameHashed(), _eip712DomainSalt()
            );
        }
    }

    function _buildDomainSeparator(bytes32 hashedName, bytes32 salt) internal view returns (bytes32) {
        return keccak256(
            abi.encode(
                keccak256(
                    "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)"
                ),
                hashedName,
                keccak256(abi.encodePacked(_eip712DomainVersion())),
                block.chainid,
                address(this),
                salt
            )
        );
    }

    /**
     * @dev Given an already https://eips.ethereum.org/EIPS/eip-712#definition-of-hashstruct[hashed struct], this
     * function returns the hash of the fully encoded EIP712 message for this domain.
     *
     * This hash can be used together with {ECDSA-recover} to obtain the signer of a message. For example:
     *
     * ```solidity
     * bytes32 digest = _hashTypedDataV4(keccak256(abi.encode(
     *     keccak256("Mail(address to,string contents)"),
     *     mailTo,
     *     keccak256(bytes(mailContents))
     * )));
     * address signer = ECDSA.recover(digest, signature);
     * ```
     */
    function _hashTypedDataV4(bytes32 structHash) internal view returns (bytes32) {
        return ECDSA.toTypedDataHash(_domainSeparatorV4(), structHash);
    }

    /**
     * @dev See {IERC20Permit-permit}.
     */
    function permit(
        address owner_,
        address spender_,
        uint256 value_,
        uint256 deadline_,
        uint8 v_,
        bytes32 r_,
        bytes32 s_
    ) public {
        // for those tokens that have been initialized before permit, we need to set
        // the permit state variables if they have not been set before
        _initializePermitStateIfNeeded();

        // permit is only allowed before the signature's deadline
        require(block.timestamp <= deadline_, "ERC20Permit: expired deadline");

        bytes32 structHash = keccak256(
            abi.encode(
                keccak256(
                    "Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"
                ),
                owner_,
                spender_,
                value_,
                _useNonce(owner_),
                deadline_
            )
        );

        bytes32 message = _hashTypedDataV4(structHash);
        address signer = ECDSA.recover(message, v_, r_, s_);

        // if we cannot recover the token owner, signature is invalid
        require(signer == owner_, "ERC20Permit: invalid signature");

        _approve(owner_, spender_, value_);
    }

    /**
     * @dev See {IERC20Permit-DOMAIN_SEPARATOR}.
     */
    // solhint-disable-next-line func-name-mixedcase
    function DOMAIN_SEPARATOR() public view returns (bytes32) {
        return _domainSeparatorV4();
    }

    function eip712Domain() public view returns (
        bytes1 domainFields,
        string memory domainName,
        string memory domainVersion,
        uint256 domainChainId,
        address domainVerifyingContract,
        bytes32 domainSalt,
        uint256[] memory domainExtensions
    ) {
        return (
            hex"1F", // 11111
            name(),
            _eip712DomainVersion(),
            block.chainid,
            address(this),
            _eip712DomainSalt(),
            new uint256[](0)
        );
    }

    function _eip712DomainVersion() internal pure returns (string memory) {
        return "1";
    }

    function _eip712DomainNameHashed() internal view returns (bytes32) {
        return keccak256(abi.encodePacked(name()));
    }

    function _eip712DomainSalt() internal view returns (bytes32) {
        return keccak256(abi.encodePacked(_state.chainId, _state.nativeContract));
    }
}
