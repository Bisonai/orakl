// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, stdJson, console} from "forge-std/Script.sol";
import {UtilsScript} from "./Utils.s.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Feed} from "../src/Feed.sol";
import {strings} from "solidity-stringutils/strings.sol";
import {Strings} from "@openzeppelin/contracts/utils/Strings.sol";

contract DeployFull is Script {
    using stdJson for string;
    using strings for *;

    function setUp() public {}

    function run() public {
        UtilsScript config = new UtilsScript();
        string memory dirPath = string.concat("/migration/", config.chainName(), "/Full");
        string[] memory migrationFiles = config.loadMigration(dirPath);

        vm.startBroadcast(vm.envUint("PRIVATE_KEY"));

        for (uint256 i = 0; i < migrationFiles.length; i++) {
            string memory migrationFilePath = migrationFiles[i];
            string memory json = vm.readFile(migrationFilePath);

            SubmissionProxy sp = new SubmissionProxy();
            bytes memory oracleRaw = json.parseRaw(".submissionProxy.oracle");
            address oracle = abi.decode(oracleRaw, (address));
            sp.addOracle(oracle);
            console.log("SubmissionProxy", address(sp));

            bytes memory numFeedsRaw = json.parseRaw(".numFeeds");
            uint256 numFeeds = abi.decode(numFeedsRaw, (uint256));

            for (uint256 j = 0; j < numFeeds; j++) {
                buildFeed(json, j, address(sp), oracle);
            }
            config.updateMigration(dirPath, migrationFilePath);
        }

        vm.stopBroadcast();
    }

    function buildFeed(string memory json, uint256 feedIndex, address submissionProxy, address oracle) internal {
        UtilsScript.FeedConstructor memory constructor_ =
            abi.decode(json.parseRaw(buildJsonQuery(feedIndex)), (UtilsScript.FeedConstructor));
        uint8 decimals = uint8(constructor_.decimals);
        string memory description = constructor_.description;

        Feed feed = new Feed(decimals, description);
        address[] memory remove_;
        address[] memory add_ = new address[](2);
        add_[0] = submissionProxy;
	add_[1] = oracle;
        feed.changeOracles(remove_, add_);

        console.log(description, address(feed));
    }

    function buildJsonQuery(uint256 index) internal returns (string memory) {
        string memory first = ".feed[";
        string memory second = Strings.toString(index);
        string memory third = "].constructor";

        return first.toSlice().concat(second.toSlice()).toSlice().concat(third.toSlice()).toSlice().toString();
    }
}
