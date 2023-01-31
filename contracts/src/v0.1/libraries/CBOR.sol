// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import 'hardhat/console.sol';
// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/vendor/CBORChainlink.sol

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

  function encodeFixedNumeric(
      Buffer.buffer memory buf,
      uint8 major,
      uint64 value
  )
      private /*pure*/ view
  {
    console.log('encodeFixedNumeric buf.buf %s', string(buf.buf));
    console.log('encodeFixedNumeric buf.buf.length in %s', buf.buf.length);
    console.log('encodeFixedNumeric major %s', major);
    console.log('encodeFixedNumeric value %s', value);

    if (value <= 23) {
      buf.appendUint8(uint8((major << 5) | value));
      console.log('encodeFixedNumeric value <= 23');
    } else if (value <= 0xFF) {
      buf.appendUint8(uint8((major << 5) | 24));
      buf.appendInt(value, 1);
    } else if (value <= 0xFFFF) {
      buf.appendUint8(uint8((major << 5) | 25));
      buf.appendInt(value, 2);
    } else if (value <= 0xFFFFFFFF) {
      buf.appendUint8(uint8((major << 5) | 26));
      buf.appendInt(value, 4);
    } else {
      buf.appendUint8(uint8((major << 5) | 27));
      buf.appendInt(value, 8);
    }

    console.log('encodeFixedNumeric2 buf.buf %s', string(buf.buf));
    console.log('encodeFixedNumeric2 buf.buf.length in %s', buf.buf.length);
  }

  function encodeIndefiniteLengthType(Buffer.buffer memory buf, uint8 major) private /*pure*/ view {
    buf.appendUint8(uint8((major << 5) | 31));
  }

  function encodeUInt(Buffer.buffer memory buf, uint value) internal /*pure*/ view {
    if (value > 0xFFFFFFFFFFFFFFFF) {
      encodeBigNum(buf, value);
    } else {
      encodeFixedNumeric(buf, MAJOR_TYPE_INT, uint64(value));
    }
  }

  function encodeInt(Buffer.buffer memory buf, int value) internal /*pure*/ view {
    if (value < -0x10000000000000000) {
      encodeSignedBigNum(buf, value);
    } else if (value > 0xFFFFFFFFFFFFFFFF) {
      encodeBigNum(buf, uint(value));
    } else if (value >= 0) {
      encodeFixedNumeric(buf, MAJOR_TYPE_INT, uint64(uint256(value)));
    } else {
      encodeFixedNumeric(buf, MAJOR_TYPE_NEGATIVE_INT, uint64(uint256(-1 - value)));
    }
  }

  function encodeBytes(Buffer.buffer memory buf, bytes memory value) internal /*pure*/ view {
    encodeFixedNumeric(buf, MAJOR_TYPE_BYTES, uint64(value.length));
    buf.append(value);
  }

  function encodeBigNum(Buffer.buffer memory buf, uint value) internal /*pure*/ view {
    buf.appendUint8(uint8((MAJOR_TYPE_TAG << 5) | TAG_TYPE_BIGNUM));
    encodeBytes(buf, abi.encode(value));
  }

  function encodeSignedBigNum(Buffer.buffer memory buf, int input) internal /*pure*/ view {
    buf.appendUint8(uint8((MAJOR_TYPE_TAG << 5) | TAG_TYPE_NEGATIVE_BIGNUM));
    encodeBytes(buf, abi.encode(uint256(-1 - input)));
  }

  function encodeString(Buffer.buffer memory buf, string memory value) internal view /*pure*/ {
    console.log('encodeString buf.capacity %s', buf.capacity);
    console.log('encodeString buf.buf.length %s', buf.buf.length);
    console.log('encodeString string(buf.buf) %s ', string(buf.buf));
    console.log('encodeString bytes(value).length %s ', bytes(value).length);
    console.log('encodeString value %s ', value);
    /* console.log('encodeString uint64(bytes(value).length %s', uint64(bytes(value).length)); */

    /* encodeFixedNumeric(buf, MAJOR_TYPE_STRING, uint64(bytes(value).length)); */
    encodeFixedNumeric(buf, MAJOR_TYPE_STRING, 1);

    console.log('encodeString2 buf.capacity %s', buf.capacity);
    console.log('encodeString2 buf.buf.length %s', buf.buf.length);
    console.log('encodeString2 string(buf.buf) %s ', string(buf.buf));
    console.log('encodeString2 bytes(value).length %s ', bytes(value).length);
    /* console.log('encodeString2 value %s ', value); */

    buf.append(bytes(value));
  }

  function startArray(Buffer.buffer memory buf) internal /*pure*/ view {
    encodeIndefiniteLengthType(buf, MAJOR_TYPE_ARRAY);
  }

  function startMap(Buffer.buffer memory buf) internal /*pure*/ view {
    encodeIndefiniteLengthType(buf, MAJOR_TYPE_MAP);
  }

  function endSequence(Buffer.buffer memory buf) internal /*pure*/ view {
    encodeIndefiniteLengthType(buf, MAJOR_TYPE_CONTENT_FREE);
  }
}
