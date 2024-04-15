// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Feed} from "../src/Feed.sol";
import {FeedProxy} from "../src/FeedProxy.sol";

contract FeedProxyTest is Test {
    Feed public feed;
    FeedProxy public feedProxy;

    address oracle = makeAddr("oracle");
    uint8 decimals = 18;
    string description = "Test Feed";

    function setUp() public {
        feed = new Feed(decimals, description, oracle);
        feedProxy = new FeedProxy(address(feed));
    }

    function test_revertWithNoDataPresent() public {
        vm.expectRevert(Feed.NoDataPresent.selector);
        feedProxy.latestRoundData();
    }

    function test_readLatestRoundData() public {
        int256 answer_ = 10;

        vm.prank(oracle);
        feed.submit(answer_);

        (uint80 latestRoundId_, int256 latestAnswer_, uint256 latestUpdatedAt_) = feedProxy.latestRoundData();
        assertEq(latestRoundId_, 1);
        assertEq(latestAnswer_, answer_);
        assertEq(latestUpdatedAt_, block.timestamp);
    }

    // | time   | 16 | 31 | 46 | 61 | 76 |
    // | answer | 10 | 20 | 30 | 40 | 50 |
    function test_twapWithoutRestrictions() public {
        uint256 heartbeat_ = 15;

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(10);

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(20);

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(30);

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(40);

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(50);

        // TWAP in the last 60 seconds
        int256 twap = feedProxy.twap(60, 0, 0);

        // (10 + 20 + 30 + 40 + 50) / 5 = 30
        assertEq(twap, 30);
    }

    function test_twapAnswerAboveTolerance() public {
        vm.prank(oracle);
        feed.submit(10);

        uint256 heartbeat_ = 10;
        vm.warp(block.timestamp + heartbeat_);

        vm.expectRevert(FeedProxy.AnswerAboveTolerance.selector);
        feedProxy.twap(60, heartbeat_ / 2, 0);
    }

    function test_twapInsufficientData() public {
        vm.prank(oracle);
        feed.submit(10);

        vm.expectRevert(FeedProxy.InsufficientData.selector);
        feedProxy.twap(60, 0, 0);
    }

    // | time   | 16 | 31 | 46 | 61 | 76 |
    // | answer | 10 | 20 | 30 | 40 | 50 |
    function test_twapWithMinCount() public {
        uint256 heartbeat_ = 15;

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(10);

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(20);

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(30);

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(40);

        vm.warp(block.timestamp + heartbeat_);
        vm.prank(oracle);
        feed.submit(50);

        // TWAP in the last 5 seconds with at least three data points
        int256 twap = feedProxy.twap(5, 0, 3);

        // (30 + 40 + 50) / 3 = 40
        assertEq(twap, 40);
    }
}
