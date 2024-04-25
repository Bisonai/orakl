// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Test, console} from "forge-std/Test.sol";
import {FeedRouter} from "../src/FeedRouter.sol";

contract FeedRouterTest is Test {
    FeedRouter public feedRouter;

    event ProxyAdded(string feedName, address indexed proxyAddress);
    event ProxyRemoved(string feedName, address indexed proxyAddress);
    event ProxyUpdated(string feedName, address indexed proxyAddress);

    error OwnableUnauthorizedAccount(address account);

    function setUp() public {
        feedRouter = new FeedRouter();
    }

    function test_AddProxy() public {
        address nonOwner_ = makeAddr("non-owner");
        string[] memory feedNames_ = new string[](2);
        address[] memory proxyAddresses_ = new address[](2);

        address btcUsdt = makeAddr("btc-usdt");
        address ethUsdt = makeAddr("eth-usdt");

        feedNames_[0] = "btc-usdt";
        feedNames_[1] = "eth-usdt";

        proxyAddresses_[0] = btcUsdt;
        proxyAddresses_[1] = ethUsdt;

        // FAIL - updateProxyBulk can be called only by owner
        vm.prank(nonOwner_);
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner_));
        feedRouter.updateProxyBulk(feedNames_, proxyAddresses_);

        vm.expectEmit(true, true, true, true);
        emit ProxyAdded(feedNames_[0], proxyAddresses_[0]);
        vm.expectEmit(true, true, true, true);
        emit ProxyAdded(feedNames_[1], proxyAddresses_[1]);
        feedRouter.updateProxyBulk(feedNames_, proxyAddresses_);

        assertEq(feedRouter.feedToProxies(feedNames_[0]), btcUsdt);
        assertEq(feedRouter.feedToProxies(feedNames_[1]), ethUsdt);
        assertEq(true, compareArrays(feedNames_, feedRouter.getFeedNames()));
    }

    function test_UpdateProxy() public {
        string[] memory feedNames_ = new string[](1);
        address[] memory proxyAddressesOld_ = new address[](1);
        address[] memory proxyAddressesNew_ = new address[](1);

        address btcUsdtOld = makeAddr("btc-usdt-old");
        address btcUsdtNew = makeAddr("btc-usdt-new");
        feedNames_[0] = "btc-usdt";

        proxyAddressesOld_[0] = btcUsdtOld;
        proxyAddressesNew_[0] = btcUsdtNew;

        vm.expectEmit(true, true, true, true);
        emit ProxyAdded(feedNames_[0], proxyAddressesOld_[0]);
        feedRouter.updateProxyBulk(feedNames_, proxyAddressesOld_);
        assertEq(btcUsdtOld, feedRouter.feedToProxies(feedNames_[0]));
        assertEq(feedRouter.feedToProxies(feedNames_[0]), btcUsdtOld);

        vm.expectEmit(true, true, true, true);
        emit ProxyUpdated(feedNames_[0], proxyAddressesNew_[0]);
        feedRouter.updateProxyBulk(feedNames_, proxyAddressesNew_);
        assertEq(btcUsdtNew, feedRouter.feedToProxies(feedNames_[0]));
        assertEq(feedRouter.feedToProxies(feedNames_[0]), btcUsdtNew);
    }

    function test_UpdateNotEqualLength() public {
        string[] memory feedNames_ = new string[](2);
        address[] memory proxyAddresses_ = new address[](1);

        // FAIL - feedNames_ is longer than proxyAddresses_
        vm.expectRevert(FeedRouter.InvalidInput.selector);
        feedRouter.updateProxyBulk(feedNames_, proxyAddresses_);
    }

    function test_UpdateInvalidProxy() public {
        string[] memory feedNames_ = new string[](1);
        address[] memory proxyAddresses_ = new address[](1);

        feedNames_[0] = "btc-usdt";
        proxyAddresses_[0] = address(0);

        // FAIL - proxy address is 0
        vm.expectRevert(FeedRouter.InvalidProxyAddress.selector);
        feedRouter.updateProxyBulk(feedNames_, proxyAddresses_);
    }

    function test_RemoveProxy() public {
        address nonOwner_ = makeAddr("non-owner");
        string[] memory feedNamesAdd_ = new string[](2);
        address[] memory proxyAddresses_ = new address[](2);

        address btcUsdt = makeAddr("btc-usdt");
        address ethUsdt = makeAddr("eth-usdt");

        feedNamesAdd_[0] = "btc-usdt";
        feedNamesAdd_[1] = "eth-usdt";

        proxyAddresses_[0] = btcUsdt;
        proxyAddresses_[1] = ethUsdt;

        // add proxies
        vm.expectEmit(true, true, true, true);
        emit ProxyAdded(feedNamesAdd_[0], proxyAddresses_[0]);
        feedRouter.updateProxyBulk(feedNamesAdd_, proxyAddresses_);
        assertEq(feedRouter.feedToProxies(feedNamesAdd_[0]), btcUsdt);
        assertEq(feedRouter.feedToProxies(feedNamesAdd_[1]), ethUsdt);

        // FAIL - cannot call removeProxyBulk with empty array
        string[] memory feedNamesEmpty_;
        vm.expectRevert(FeedRouter.InvalidInput.selector);
        feedRouter.removeProxyBulk(feedNamesEmpty_);

        // remove btc-usdt proxy
        string[] memory feedNamesRemove_ = new string[](1);
        feedNamesRemove_[0] = feedNamesAdd_[0];

        // FAIL - removeProxyBulk can be called only by owner
        vm.prank(nonOwner_);
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, nonOwner_));
        feedRouter.removeProxyBulk(feedNamesRemove_);

        vm.expectEmit(true, true, true, true);
        emit ProxyRemoved(feedNamesRemove_[0], proxyAddresses_[0]);
        feedRouter.removeProxyBulk(feedNamesRemove_);

        // expect that only eth-usdt is left
        string[] memory expectedFeedNames_ = new string[](1);
        expectedFeedNames_[0] = feedNamesAdd_[1];
        assertEq(true, compareArrays(expectedFeedNames_, feedRouter.getFeedNames()));

        assertEq(feedRouter.feedToProxies(feedNamesAdd_[0]), address(0));
        assertEq(feedRouter.feedToProxies(feedNamesAdd_[1]), ethUsdt);
    }

    function test_GetDataFromNonExistingFeed() public {
        // FAIL - cannot read data from feed that has not been set
        vm.expectRevert(FeedRouter.FeedNotSetInRouter.selector);
        feedRouter.latestRoundData("NOT-EXIST");
    }

    function compareArrays(string[] memory array1, string[] memory array2) private pure returns (bool) {
        if (array1.length != array2.length) {
            return false;
        }

        for (uint256 i = 0; i < array1.length; i++) {
            bool found = false;
            for (uint256 j = 0; j < array2.length; j++) {
                if (keccak256(abi.encodePacked(array1[i])) == keccak256(abi.encodePacked(array2[j]))) {
                    found = true;
                    break;
                }
            }
            if (!found) {
                return false;
            }
        }

        return true;
    }
}
