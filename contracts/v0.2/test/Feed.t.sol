// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Feed} from "../src/Feed.sol";

contract FeedTest is Test {
    Feed public feed;
    uint8 decimals = 18;
    string description = "Test Feed";
    address[] removed;
    address[] added;

    event FeedUpdated(int256 indexed answer, uint256 indexed roundId, uint256 updatedAt);

    function setUp() public {
        feed = new Feed(decimals, description);
    }

    function test_AddAndRemoveOracle() public {
	address alice = makeAddr('alice');
	address bob = makeAddr('bob');

        added.push(alice);
        added.push(bob);

	feed.changeOracles(removed, added);
        assertEq(feed.getOracles().length, 2);

	// remove what has been added (switched parameters)
	feed.changeOracles(added, removed);
        assertEq(feed.getOracles().length, 0);
    }

    function test_RemoveNonexistantOracle() public {
	address alice = makeAddr('alice');

	removed.push(alice);
	vm.expectRevert(Feed.OracleNotEnabled.selector);
	feed.changeOracles(removed, added);
    }

    function test_AddOracleTwice() public {
	address alice = makeAddr('alice');

	added.push(alice);
	feed.changeOracles(removed, added);
	vm.expectRevert(Feed.OracleAlreadyEnabled.selector);
	feed.changeOracles(removed, added);
    }

    function test_SubmitAndReadResponse() public {
	address alice = makeAddr('alice');
	added.push(alice);
	feed.changeOracles(removed, added);

	int256 expectedAnswer = 10;
	uint256 expectedRoundId = 1;
	uint256 expectedUpdatedAt = block.timestamp;

        vm.expectEmit(true, true, true, true);
        emit FeedUpdated(expectedAnswer, expectedRoundId, expectedUpdatedAt);
        feed.submit(expectedAnswer);
        (uint80 roundId, int256 answer, uint256 updatedAt) = feed.latestRoundData();
	assertEq(roundId, expectedRoundId);
        assertEq(answer, expectedAnswer);
	assertEq(updatedAt, expectedUpdatedAt);
    }
}
