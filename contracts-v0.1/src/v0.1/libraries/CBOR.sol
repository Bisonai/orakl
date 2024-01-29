// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/vendor/CBORChainlink.sol

import {Buffer} from "./Buffer.sol";

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

    function encodeFixedNumeric(Buffer.buffer memory buf, uint8 major, uint64 value) private pure {
        if (value <= 23) {
            buf.appendUint8(uint8((major << 5) | value));
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
    }

    function encodeIndefiniteLengthType(Buffer.buffer memory buf, uint8 major) private pure {
        buf.appendUint8(uint8((major << 5) | 31));
    }

    function encodeUInt(Buffer.buffer memory buf, uint value) internal pure {
        if (value > 0xFFFFFFFFFFFFFFFF) {
            encodeBigNum(buf, value);
        } else {
            encodeFixedNumeric(buf, MAJOR_TYPE_INT, uint64(value));
        }
    }

    function encodeInt(Buffer.buffer memory buf, int value) internal pure {
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

    function encodeBytes(Buffer.buffer memory buf, bytes memory value) internal pure {
        encodeFixedNumeric(buf, MAJOR_TYPE_BYTES, uint64(value.length));
        buf.append(value);
    }

    function encodeBigNum(Buffer.buffer memory buf, uint value) internal pure {
        buf.appendUint8(uint8((MAJOR_TYPE_TAG << 5) | TAG_TYPE_BIGNUM));
        encodeBytes(buf, abi.encode(value));
    }

    function encodeSignedBigNum(Buffer.buffer memory buf, int input) internal pure {
        buf.appendUint8(uint8((MAJOR_TYPE_TAG << 5) | TAG_TYPE_NEGATIVE_BIGNUM));
        encodeBytes(buf, abi.encode(uint256(-1 - input)));
    }

    function encodeString(Buffer.buffer memory buf, string memory value) internal pure {
        encodeFixedNumeric(buf, MAJOR_TYPE_STRING, uint64(bytes(value).length));
        buf.append(bytes(value));
    }

    function startArray(Buffer.buffer memory buf) internal pure {
        encodeIndefiniteLengthType(buf, MAJOR_TYPE_ARRAY);
    }

    function startMap(Buffer.buffer memory buf) internal pure {
        encodeIndefiniteLengthType(buf, MAJOR_TYPE_MAP);
    }

    function endSequence(Buffer.buffer memory buf) internal pure {
        encodeIndefiniteLengthType(buf, MAJOR_TYPE_CONTENT_FREE);
    }
}
