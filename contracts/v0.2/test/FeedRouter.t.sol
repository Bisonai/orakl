// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {FeedRouter} from "../src/FeedRouter.sol";

contract FeedRouterTest is Test {
    FeedRouter public feedRouter;

    function setUp() public {
        feedRouter = new FeedRouter();
    }

    function test_AddProxy() public {
	string[] memory feedNames_ = new string[](2);
	address[] memory proxyAddresses_ = new address[](2);

	address btcUsdt = makeAddr("btc-usdt");
	address ethUsdt = makeAddr("eth-usdt");

	feedNames_[0] = "btc-usdt";
	feedNames_[1] = "eth-usdt";

	proxyAddresses_[0] = btcUsdt;
	proxyAddresses_[1] = ethUsdt;

	feedRouter.updateProxyBulk(feedNames_, proxyAddresses_);

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

	feedRouter.updateProxyBulk(feedNames_, proxyAddressesOld_);
	assertEq(btcUsdtOld, feedRouter.feedToProxies(feedNames_[0]));

	feedRouter.updateProxyBulk(feedNames_, proxyAddressesNew_);
	assertEq(btcUsdtNew, feedRouter.feedToProxies(feedNames_[0]));
    }

    function test_UpdateNotEqualLength() public {
	string[] memory feedNames_ = new string[](2);
	address[] memory proxyAddresses_ = new address[](1);

	// feedNames_ is longer than proxyAddresses_ -> FAIL
	vm.expectRevert("invalid input");
	feedRouter.updateProxyBulk(feedNames_, proxyAddresses_);
    }

    function test_UpdateInvalidProxy() public {
	string[] memory feedNames_ = new string[](1);
	address[] memory proxyAddresses_ = new address[](1);

	feedNames_[0] = "btc-usdt";
	proxyAddresses_[0] = address(0);

	// proxy address is 0 -> FAIL
	vm.expectRevert(FeedRouter.InvalidProxyAddress.selector);
	feedRouter.updateProxyBulk(feedNames_, proxyAddresses_);
    }

    function test_RemoveProxy() public {
	string[] memory feedNamesAdd_ = new string[](2);
	address[] memory proxyAddresses_ = new address[](2);

	address btcUsdt = makeAddr("btc-usdt");
	address ethUsdt = makeAddr("eth-usdt");

	feedNamesAdd_[0] = "btc-usdt";
	feedNamesAdd_[1] = "eth-usdt";

	proxyAddresses_[0] = btcUsdt;
	proxyAddresses_[1] = ethUsdt;

	// add proxies
	feedRouter.updateProxyBulk(feedNamesAdd_, proxyAddresses_);

	// remove btc-usdt proxy
	string[] memory feedNamesRemove_ = new string[](1);
	feedNamesRemove_[0] = "btc-usdt";
	feedRouter.removeProxyBulk(feedNamesRemove_);

	// expect that only eth-usdt is left
	string[] memory expectedFeedNames_ = new string[](1);
	expectedFeedNames_[0] = "eth-usdt";
	assertEq(true, compareArrays(expectedFeedNames_, feedRouter.getFeedNames()));
    }

    function compareArrays(string[] memory array1, string[] memory array2) private pure returns (bool) {
        if (array1.length != array2.length) {
            return false;
        }

        for (uint i = 0; i < array1.length; i++) {
            bool found = false;
            for (uint j = 0; j < array2.length; j++) {
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
