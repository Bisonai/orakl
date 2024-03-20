// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

abstract contract ITypeAndVersion {
    /**
     * @notice Return the type and version of contract
     * @return typeAndVersion The type and version of the contract.
     */
    function typeAndVersion() external pure virtual returns (string memory);
}
