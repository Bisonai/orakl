// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/Chainlink.sol

import {Buffer} from "./Buffer.sol";
import {CBOR} from "./CBOR.sol";

library Orakl {
    uint256 internal constant defaultBufferSize = 256;

    using CBOR for Buffer.buffer;

    // structure for storing requests done off-chain
    struct Request {
        bytes32 id;
        address callbackAddr;
        bytes4 callbackFunc;
        uint256 nonce;
        Buffer.buffer buf;
    }

    /**
     * @notice Initializes a request
     * @dev Sets ID, callback address, and callback function
     * @param self The uninitialized request
     * @param jobId The Job Specification ID
     * @param callbackAddr The callback address
     * @param callbackFunc The callback function signature
     * @return The initialized request
     */
    function initialize(
        Request memory self,
        bytes32 jobId,
        address callbackAddr,
        bytes4 callbackFunc
    ) internal pure returns (Orakl.Request memory) {
        Buffer.init(self.buf, defaultBufferSize);
        self.id = jobId;
        self.callbackAddr = callbackAddr;
        self.callbackFunc = callbackFunc;
        return self;
    }

    /**
     * @notice sets the data for buffer
     * @param _request the initialized request
     * @param _data the CBOR data
     */
    function setBuffer(Request memory _request, bytes memory _data) internal pure {
        Buffer.init(_request.buf, _data.length);
        Buffer.append(_request.buf, _data);
    }

    /**
     * @notice Adds a string value to the request in a key - value pair format
     * @param self - the initalized request
     * @param key - the name of the key
     * @param value - the string value to add
     */
    function add(Request memory self, string memory key, string memory value) internal pure {
        self.buf.encodeString(key);
        self.buf.encodeString(value);
    }

    /**
     * @notice Adds a byte value to the request in a key - value pair format
     * @param _request - the initalized request
     * @param _key - the name of the key
     * @param _value - the bytes value to add
     */
    function addBytes(
        Request memory _request,
        string memory _key,
        bytes memory _value
    ) internal pure {
        _request.buf.encodeString(_key);
        _request.buf.encodeBytes(_value);
    }

    /**
     * @notice Adds a Int256 value to the request in a key - value pair format
     * @param _request - the initalized request
     * @param _key - the name of the key
     * @param _value - the int256 value to add
     */
    function addInt(Request memory _request, string memory _key, int256 _value) internal pure {
        _request.buf.encodeString(_key);
        _request.buf.encodeInt(_value);
    }

    /**
     * @notice Adds a UInt256 value to the request in a key - value pair format
     * @param _request - the initalized request
     * @param _key - the name of the key
     * @param _value - the uint256 value to add
     */
    function addUInt(Request memory _request, string memory _key, uint256 _value) internal pure {
        _request.buf.encodeString(_key);
        _request.buf.encodeUInt(_value);
    }

    /**
     * @notice Adds an array of string value to the request in a key - value pair format
     * @param _request - the initalized request
     * @param _key - the name of the key
     * @param _values - the array of string value to add
     */
    function addStringArray(
        Request memory _request,
        string memory _key,
        string[] memory _values
    ) internal pure {
        _request.buf.encodeString(_key);
        _request.buf.startArray();
        for (uint256 i; i < _values.length; i++) {
            _request.buf.encodeString(_values[i]);
        }
        _request.buf.endSequence();
    }
}
