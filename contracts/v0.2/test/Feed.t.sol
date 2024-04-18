// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Feed} from "../src/Feed.sol";

contract FeedTest is Test {
    Feed public feed;

    address oracle = makeAddr("oracle");
    uint8 decimals = 18;
    string description = "Test Feed";

    event FeedUpdated(int256 indexed answer, uint256 indexed roundId, uint256 updatedAt);
    event SubmitterUpdated(address indexed submitter);
    error OwnableUnauthorizedAccount(address account);

    function setUp() public {
        feed = new Feed(decimals, description, oracle);
    }

    function test_UpdateSubmitter() public {
	address newSubmitter = makeAddr("new-submitter");
	assert(feed.submitter() != newSubmitter);

	// SUCCESS
	vm.expectEmit(true, true, true, true);
        emit SubmitterUpdated(newSubmitter);
	feed.updateSubmitter(newSubmitter);
	assertEq(feed.submitter(), newSubmitter);
    }

    function test_UpdateSubmitterWithNonOwner() public {
	address nonOwner = makeAddr("non-owner");
	address newSubmitter = makeAddr("new-submitter");

	// FAIL - only owner can update submitter
	vm.prank(nonOwner);
	vm.expectRevert(
	    abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner)
	);
	feed.updateSubmitter(newSubmitter);
    }

    function test_UpdateSubmitterWithZeroAddress() public {
	// FAIL - cannot set submitter to address(0)
	vm.expectRevert(Feed.InvalidSubmitter.selector);
	feed.updateSubmitter(address(0));
    }

    function test_SubmitAndReadResponse() public {
        int256 expectedAnswer_ = 10;
        uint256 expectedRoundId_ = 1;
        uint256 expectedUpdatedAt_ = block.timestamp;

        vm.prank(oracle);
        vm.expectEmit(true, true, true, true);
        emit FeedUpdated(expectedAnswer_, expectedRoundId_, expectedUpdatedAt_);
        feed.submit(expectedAnswer_);
        (uint80 roundId, int256 answer, uint256 updatedAt) = feed.latestRoundData();
        assertEq(roundId, expectedRoundId_);
        assertEq(answer, expectedAnswer_);
        assertEq(updatedAt, expectedUpdatedAt_);
    }
}
