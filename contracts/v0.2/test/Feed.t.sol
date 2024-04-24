// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Feed} from "../src/Feed.sol";

contract FeedTest is Test {
    Feed public feed;

    address submitter = makeAddr("submitter");
    uint8 decimals = 18;
    string description = "Test Feed";

    event FeedUpdated(int256 indexed answer);
    event SubmitterUpdated(address indexed submitter);

    error OwnableUnauthorizedAccount(address account);

    function setUp() public {
        feed = new Feed(decimals, description, submitter);
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
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner));
        feed.updateSubmitter(newSubmitter);
    }

    function test_UpdateSubmitterWithZeroAddress() public {
        // FAIL - cannot set submitter to address(0)
        vm.expectRevert(Feed.InvalidSubmitter.selector);
        feed.updateSubmitter(address(0));
    }

    function test_SubmitAndReadResponse() public {
        int256 expectedAnswer_ = 10;

        vm.prank(submitter);
        vm.expectEmit(true, true, true, true);
        emit FeedUpdated(expectedAnswer_);
        feed.submit(expectedAnswer_);
        (, int256 answer,) = feed.latestRoundData();
        assertEq(answer, expectedAnswer_);
    }

    function test_SubmitByNonSubmitter() public {
        address nonSubmitter = makeAddr("non-submitter");

	// only submitter is allowed to submit
        vm.prank(nonSubmitter);
	vm.expectRevert(Feed.OnlySubmitter.selector);
        feed.submit(10);
    }
}
