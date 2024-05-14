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


    function setUp() public {}

    function run() public {
        UtilsScript config = new UtilsScript();
        string memory dirPath = string.concat("/migration/", config.chainName(), "/Feed");
        string[] memory migrationFiles = config.loadMigration(dirPath);

        vm.startBroadcast(vm.envUint("PRIVATE_KEY"));

        for (uint256 i = 0; i < migrationFiles.length; i++) {
            string memory migrationFilePath = migrationFiles[i];
            string memory json = vm.readFile(migrationFilePath);
            console.log("Migration File", migrationFilePath);
            bool deploy = vm.keyExists(json, ".deploy");

            if (deploy) {
                console.log("Deploying Feed");
                bytes memory submitterRaw = json.parseRaw(".deploy.submitter");
                bytes memory feedNamesRaw = json.parseRaw(".deploy.feedNames");
                address submitter = abi.decode(submitterRaw, (address));
                string[] memory feedNames = abi.decode(feedNamesRaw, (string[]));
                for (uint256 j = 0; j < feedNames.length; j++) {
                    Feed feed = new Feed(DECIMALS, feedNames[j], submitter);
                    console.log("(Feed Deployed)", feedNames[j], address(feed));
                    FeedProxy feedProxy = new FeedProxy(address(feed));
                    console.log("(FeedProxy Deployed)", feedNames[j], address(feedProxy));
                }
            }

            bool updateSubmitter = vm.keyExists(json, ".updateSubmitter");
            if (updateSubmitter) {
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

            config.updateMigration(dirPath, migrationFilePath);
        }
        vm.stopBroadcast();
    }

}