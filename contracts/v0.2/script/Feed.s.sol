// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script, stdJson, console} from "forge-std/Script.sol";
import {UtilsScript} from "./Utils.s.sol";
import {Feed} from "../src/Feed.sol";
import {FeedProxy} from "../src/FeedProxy.sol";
import {strings} from "solidity-stringutils/strings.sol";
import {Strings} from "@openzeppelin/contracts/utils/Strings.sol";

contract DeployFeed is Script {
    using stdJson for string;
    using strings for *;

    uint8 constant DECIMALS = 8;
    UtilsScript config;

    function setUp() public {
        config = new UtilsScript();
    }

    function run() public {
        string memory dirPath = string.concat("/migration/", config.chainName(), "/Feed");
        string[] memory migrationFiles = config.loadMigration(dirPath);

        vm.startBroadcast(vm.envUint("PRIVATE_KEY"));

        for (uint256 i = 0; i < migrationFiles.length; i++) {
            string memory migrationFilePath = migrationFiles[i];
            string memory json = vm.readFile(migrationFilePath);
            console.log("Migration File", migrationFilePath);
            executeMigration(json);
            config.updateMigration(dirPath, migrationFilePath);
        }
        vm.stopBroadcast();
    }

    function executeMigration(string memory json) public {
        console.log("Executing Migration");

        deployFeeds(json);
        updateSubmitter(json);
        proposeFeeds(json);
        confirmFeeds(json);
    }

    function deployFeeds(string memory json) internal {
        if (!vm.keyExists(json, ".deploy")) {
            return;
        }
        console.log("Deploying Feed");
        bytes memory submitterRaw = json.parseRaw(".deploy.submitter");
        bytes memory feedNamesRaw = json.parseRaw(".deploy.feedNames");
        address submitter = abi.decode(submitterRaw, (address));
        string[] memory feedNames = abi.decode(feedNamesRaw, (string[]));
        for (uint256 j = 0; j < feedNames.length; j++) {
            // TODO: support general decimals setting
            uint8 decimals = DECIMALS;
            if (compareStrings(feedNames[j], "BABYDOGE-USDT")) {
                decimals = 16;
            }

            Feed feed = new Feed(decimals, feedNames[j], submitter);
            console.log("(Feed Deployed)", feedNames[j], address(feed));
            FeedProxy feedProxy = new FeedProxy(address(feed));
            console.log("(FeedProxy Deployed)", feedNames[j], address(feedProxy));

            config.storeFeedAddress(feedNames[j], address(feed), address(feedProxy));
        }
    }

    function updateSubmitter(string memory json) internal {
        if (!vm.keyExists(json, ".updateSubmitter")) {
            return;
        }
        console.log("Updating Feed Submitter");
        bytes memory submitterRaw = json.parseRaw(".updateSubmitter.submitter");
        bytes memory feedAddressesRaw = json.parseRaw(".updateSubmitter.feedAddresses");
        address submitter = abi.decode(submitterRaw, (address));
        address[] memory feedAddresses = abi.decode(feedAddressesRaw, (address[]));
        for (uint256 j = 0; j < feedAddresses.length; j++) {
            Feed feed = Feed(feedAddresses[j]);
            feed.updateSubmitter(submitter);
            string memory feedName = feed.name();
            console.log("(Submitter Updated)", feedName, submitter);
        }
    }

    function proposeFeeds(string memory json) internal {
        if (!vm.keyExists(json, ".proposeFeeds")) {
            return;
        }
        console.log("Proposing Feeds to FeedProxies");
        bytes memory raw = json.parseRaw(".proposeFeeds");
        UtilsScript.FeedProxyUpdateConstructor[] memory updateSets =
            abi.decode(raw, (UtilsScript.FeedProxyUpdateConstructor[]));
        for (uint256 j = 0; j < updateSets.length; j++) {
            UtilsScript.FeedProxyUpdateConstructor memory updateSet = updateSets[j];
            FeedProxy feedProxy = FeedProxy(updateSet.feedProxyAddress);
            feedProxy.proposeFeed(updateSet.feedAddress);
            console.log("(Feed Proposed)", updateSet.feedAddress, updateSet.feedProxyAddress);
        }
    }

    function confirmFeeds(string memory json) internal {
        if (!vm.keyExists(json, ".confirmFeeds")) {
            return;
        }
        console.log("Confirming Feeds to FeedProxies");
        bytes memory raw = json.parseRaw(".confirmFeeds");
        UtilsScript.FeedProxyUpdateConstructor[] memory updateSets =
            abi.decode(raw, (UtilsScript.FeedProxyUpdateConstructor[]));
        for (uint256 j = 0; j < updateSets.length; j++) {
            UtilsScript.FeedProxyUpdateConstructor memory updateSet = updateSets[j];
            FeedProxy feedProxy = FeedProxy(updateSet.feedProxyAddress);
            feedProxy.confirmFeed(updateSet.feedAddress);
            console.log("(Feed Confirmed)", updateSet.feedProxyAddress, updateSet.feedAddress);
        }
    }

     function compareStrings(string memory s1, string memory s2) public pure returns (bool) {
        return keccak256(abi.encodePacked(s1)) == keccak256(abi.encodePacked(s2));
    }
}
