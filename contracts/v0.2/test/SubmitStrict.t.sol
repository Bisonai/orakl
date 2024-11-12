// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Vm.sol";
import {Test, console} from "forge-std/Test.sol";
import { stdJson } from "forge-std/StdJson.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {FeedRouter} from "../src/FeedRouter.sol";
import "forge-std/console.sol";

contract SubmitStrictTest is Test {
    using stdJson for string;

    function run() public {
	string memory json = vm.readFile("submission.json");

	string[] memory symbols = getsymbols(json);
	bytes32[] memory feedHashes = gethashes(json);
	int256[] memory valuesInt = getvalues(json);
	uint256[] memory timestampsInt = gettimestamps(json);
	bytes[] memory proofs = getproofs(json);

	console.log("submission");
	console.logBytes32(feedHashes[0]);
	console.logInt(valuesInt[0]);
	console.logUint(timestampsInt[0]);
	console.logBytes(proofs[0]);

	SubmissionProxy sp = SubmissionProxy(0x3a251c738e19806A546815eb6065e139A8D65B4b);
	sp.submitStrictBatch(
	    feedHashes,
	    valuesInt,
	    timestampsInt,
	    proofs
	);

	console.log(block.timestamp);

	FeedRouter fr = FeedRouter(0x653078F0D3a230416A59aA6486466470Db0190A2);
	(uint64 roundId, int256 answer, uint256 blockTimestamp) = fr.latestRoundData(symbols[0]);

	console.log("latestRoundData");
	console.logUint(roundId);
	console.logInt(answer);
	console.logUint(blockTimestamp);

	require(block.timestamp == blockTimestamp, "Timestamps do not match");
    }

    function getsymbols(string memory json) public pure returns (string[] memory) {
	bytes memory rawSymbols = json.parseRaw(".symbols");
	return abi.decode(rawSymbols, (string[]));
    }

    function gethashes(string memory json) public pure returns (bytes32[] memory) {
	bytes memory rawFeedHashes = json.parseRaw(".feedHashes");
        return abi.decode(rawFeedHashes, (bytes32[]));
    }

    function getvalues(string memory json) public pure returns (int256[] memory) {
	bytes memory rawValues = json.parseRaw(".values");
	string[] memory values = abi.decode(rawValues, (string[]));
	int256[] memory valuesInt = new int256[](values.length);
	for (uint i = 0; i < values.length; i++) {
	    valuesInt[i] = vm.parseInt(values[i]);
	}
	return valuesInt;
    }

    function gettimestamps(string memory json) public pure returns (uint256[] memory) {
	bytes memory rawTimestamps = json.parseRaw(".aggregateTimes");
	string[] memory timestamps = abi.decode(rawTimestamps, (string[]));
	uint256[] memory timestampsInt = new uint256[](timestamps.length);
	for (uint i = 0; i < timestamps.length; i++) {
	    timestampsInt[i] = vm.parseUint(timestamps[i]);
	}
	return timestampsInt;
    }

    function getproofs(string memory json) public pure returns (bytes[] memory) {
	bytes memory rawProofs = json.parseRaw(".proofs");
	return abi.decode(rawProofs, (bytes[]));
    }
}
