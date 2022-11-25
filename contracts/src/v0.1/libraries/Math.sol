// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

/**
 * @dev Math library is a library that contains different mathematical functions for calculations
 *
 */

library Math {
  /**
   * @dev Returns max value given two numbers
   *
   */
  function max(uint256 a, uint256 b) internal pure returns (uint256) {
    if (a > b) {
      return a;
    }
    return b;
  }
}
