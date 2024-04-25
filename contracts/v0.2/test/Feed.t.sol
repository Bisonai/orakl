// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

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
        address newSubmitter_ = makeAddr("new-submitter");
        assert(feed.submitter() != newSubmitter_);

        // SUCCESS
        vm.expectEmit(true, true, true, true);
        emit SubmitterUpdated(newSubmitter_);
        feed.updateSubmitter(newSubmitter_);
        assertEq(feed.submitter(), newSubmitter_);
    }

    function test_UpdateSubmitterWithNonOwner() public {
        address nonOwner_ = makeAddr("non-owner");
        address newSubmitter_ = makeAddr("new-submitter");

        // FAIL - only owner can update submitter
        vm.prank(nonOwner_);
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner_));
        feed.updateSubmitter(newSubmitter_);
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
        (, int256 answer_,) = feed.latestRoundData();
        assertEq(answer_, expectedAnswer_);
    }

    function test_SubmitByNonSubmitter() public {
        address nonSubmitter_ = makeAddr("non-submitter");

        vm.prank(nonSubmitter_);
        // FAIL - only submitter is allowed to submit
        vm.expectRevert(Feed.OnlySubmitter.selector);
        feed.submit(10);
    }
}
