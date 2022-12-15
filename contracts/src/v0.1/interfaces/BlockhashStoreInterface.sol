// SPDX-License-Identifier: MIT
pragma solidity 0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/interfaces/BlockhashStoreInterface.sol

interface BlockhashStoreInterface {
  function getBlockhash(uint256 number) external view returns (bytes32);
}
