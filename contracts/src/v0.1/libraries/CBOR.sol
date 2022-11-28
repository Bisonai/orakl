// SPDX-License-Identifier: MIT
// Reference - https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/vendor/CBORChainlink.sol

pragma solidity ^0.8.16;

import {Buffer} from './Buffer.sol';

// Encoding library for Binary Object Representation
library CBOR {
  using Buffer for Buffer.buffer;

  // DECLARE TYPES FOR EASIER REFERENCE OF VARIABLE TYPE
  uint8 private constant MAJOR_TYPE_INT = 0;
  uint8 private constant MAJOR_TYPE_NEGATIVE_INT = 1;
  uint8 private constant MAJOR_TYPE_BYTES = 2;
  uint8 private constant MAJOR_TYPE_STRING = 3;
  uint8 private constant MAJOR_TYPE_ARRAY = 4;
  uint8 private constant MAJOR_TYPE_MAP = 5;
  uint8 private constant MAJOR_TYPE_TAG = 6;
  uint8 private constant MAJOR_TYPE_CONTENT_FREE = 7;

  uint8 private constant TAG_TYPE_BIGNUM = 2;
  uint8 private constant TAG_TYPE_NEGATIVE_BIGNUM = 3;
}
