// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/interfaces/TypeAndVersionInterface.sol

abstract contract ITypeAndVersion {
    function typeAndVersion() external pure virtual returns (string memory);
}
