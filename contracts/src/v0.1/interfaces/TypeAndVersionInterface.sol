// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/interfaces/TypeAndVersionInterface.sol

abstract contract TypeAndVersionInterface {
  function typeAndVersion() external pure virtual returns (string memory);
}
