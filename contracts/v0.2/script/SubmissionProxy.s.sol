// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script, stdJson, console} from "forge-std/Script.sol";
import {UtilsScript} from "./Utils.s.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Feed} from "../src/Feed.sol";
import {FeedProxy} from "../src/FeedProxy.sol";
import {FeedRouter} from "../src/FeedRouter.sol";
import {strings} from "solidity-stringutils/strings.sol";
import {Strings} from "@openzeppelin/contracts/utils/Strings.sol";

contract DeploySubmissionProxy is Script {
    using stdJson for string;
    using strings for *;

    uint8 constant DECIMALS = 8;
    UtilsScript config;

    function setUp() public {
        config = new UtilsScript();
    }

    function run() public {
        string memory dirPath = string.concat("/migration/", config.chainName(), "/SubmissionProxy");
        string[] memory migrationFiles = config.loadMigration(dirPath);

        vm.startBroadcast(vm.envUint("PRIVATE_KEY"));

        for (uint256 i = 0; i < migrationFiles.length; i++) {
            bool result = executeMigration(migrationFiles[i]);
            if (!result) {
                console.log("Migration failed");
                continue;
            }
            config.updateMigration(dirPath, migrationFiles[i]);
        }
        vm.stopBroadcast();
    }

    function executeMigration(string memory migrationFilePath) public returns (bool) {
        SubmissionProxy submissionProxy;
        string memory json = vm.readFile(migrationFilePath);
        console.log("Migration File", migrationFilePath);
        bool useExisting = vm.keyExists(json, ".address");
        bool deploy = vm.keyExists(json, ".deploy");

        if (deploy) {
            submissionProxy = deploySubmissionProxy();
            config.storeAddress("SubmissionProxy", address(submissionProxy));
        } else if (useExisting) {
            submissionProxy = useExistingSubmissionProxy(json);
        } else {
            console.log("SubmissionProxy not found, skipping deploy");
            return false;
        }

        setMaxSubmission(submissionProxy, json);
        setDataFreshness(submissionProxy, json);
        setExpirationPeriod(submissionProxy, json);
        setDefaultProofThreshold(submissionProxy, json);
        setProofThreshold(submissionProxy, json);
        addOracle(submissionProxy, json);
        removeOracle(submissionProxy, json);
        updateFeed(submissionProxy, json);
        removeFeed(submissionProxy, json);
        deployFeed(submissionProxy, json);

        return true;
    }

    function deploySubmissionProxy() internal returns (SubmissionProxy) {
        console.log("Deploying SubmissionProxy");
        SubmissionProxy submissionProxy = new SubmissionProxy();
        console.log("(SubmissionProxy Deployed)", address(submissionProxy));
        return submissionProxy;
    }

    function useExistingSubmissionProxy(string memory json) internal view returns (SubmissionProxy) {
        bytes memory submissionProxyAddressRaw = json.parseRaw(".address");
        address submissionProxyAddress = abi.decode(submissionProxyAddressRaw, (address));
        SubmissionProxy submissionProxy = SubmissionProxy(submissionProxyAddress);
        console.log("(Use existing SubmissionProxy)", address(submissionProxy));
        return submissionProxy;
    }

    function useExistingFeedRouter(string memory json) internal view returns (FeedRouter) {
        bytes memory feedRouterAddressRaw = json.parseRaw(".feedRouter.address");
        address feedRouterAddress = abi.decode(feedRouterAddressRaw, (address));
        FeedRouter feedRouter = FeedRouter(feedRouterAddress);
        console.log("(Use existing FeedRouter)", address(feedRouter));
        return feedRouter;

    }

    function setMaxSubmission(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".setMaxSubmission")) {
            return;
        }
        bytes memory raw = json.parseRaw(".setMaxSubmission");
        uint256 maxSubmission = abi.decode(raw, (uint256));
        submissionProxy.setMaxSubmission(maxSubmission);
        console.log("(Max Submission Set)", maxSubmission);
    }

    function setDataFreshness(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".setDataFreshness")) {
            return;
        }
        bytes memory raw = json.parseRaw(".setDataFreshness");
        uint256 dataFreshness = abi.decode(raw, (uint256));
        submissionProxy.setDataFreshness(dataFreshness);
        console.log("(Data Freshness Set)", dataFreshness);
    }

    function setExpirationPeriod(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".setExpirationPeriod")) {
            return;
        }
        bytes memory raw = json.parseRaw(".setExpirationPeriod");
        uint256 expirationPeriod = abi.decode(raw, (uint256));
        submissionProxy.setExpirationPeriod(expirationPeriod);
        console.log("(Expiration Period Set)", expirationPeriod);
    }

    function setDefaultProofThreshold(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".setDefaultProofThreshold")) {
            return;
        }
        bytes memory raw = json.parseRaw(".setDefaultProofThreshold");
        uint8 defaultProofThreshold = abi.decode(raw, (uint8));
        submissionProxy.setDefaultProofThreshold(defaultProofThreshold);
        console.log("(Default Proof Threshold Set)", defaultProofThreshold);
    }

    function setProofThreshold(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".setProofThreshold")) {
            return;
        }
        bytes memory raw = json.parseRaw(".setProofThreshold");
        UtilsScript.SetProofThresholdConstructor[] memory thresholds =
            abi.decode(raw, (UtilsScript.SetProofThresholdConstructor[]));
        for (uint256 j = 0; j < thresholds.length; j++) {
            submissionProxy.setProofThreshold(string2bytes32Hash(thresholds[j].name), thresholds[j].threshold);
            console.log("(Proof Threshold Set)", thresholds[j].name, thresholds[j].threshold);
        }
    }

    function addOracle(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".addOracle")) {
            return;
        }
        bytes memory raw = json.parseRaw(".addOracle.oracles");
        address[] memory oracles = abi.decode(raw, (address[]));
        for (uint256 j = 0; j < oracles.length; j++) {
            submissionProxy.addOracle(oracles[j]);
            console.log("(Oracle Added)", oracles[j]);
        }
    }

    function removeOracle(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".removeOracle")) {
            return;
        }
        bytes memory raw = json.parseRaw(".removeOracle.oracles");
        address[] memory oracles = abi.decode(raw, (address[]));
        for (uint256 j = 0; j < oracles.length; j++) {
            submissionProxy.removeOracle(oracles[j]);
            console.log("(Oracle Removed)", oracles[j]);
        }
    }

    function updateFeed(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".updateFeed")) {
            return;
        }
        bytes memory raw = json.parseRaw(".updateFeed");
        _updateFeeds(submissionProxy, raw);
    }

    function _updateFeeds(SubmissionProxy submissionProxy, bytes memory raw) internal {
        UtilsScript.UpdateFeedConstructor[] memory feeds = abi.decode(raw, (UtilsScript.UpdateFeedConstructor[]));
        bytes32[] memory feedHashes = new bytes32[](feeds.length);
        address[] memory feedAddresses = new address[](feeds.length);
        for (uint256 j = 0; j < feeds.length; j++) {
            UtilsScript.UpdateFeedConstructor memory feed = feeds[j];
            feedHashes[j] = string2bytes32Hash(feed.name);
            feedAddresses[j] = feed.feedAddress;

            console.log("(Feed Prepared)", feed.name, feed.feedAddress);
        }
        submissionProxy.updateFeedBulk(feedHashes, feedAddresses);
    }

    function removeFeed(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".removeFeed")) {
            return;
        }
        bytes memory raw = json.parseRaw(".removeFeed.feedNames");
        string[] memory feedNames = abi.decode(raw, (string[]));
        for (uint256 j = 0; j < feedNames.length; j++) {
            submissionProxy.removeFeed(string2bytes32Hash(feedNames[j]));
            console.log("(Feed Removed)", feedNames[j]);
        }
    }

    function deployFeed(SubmissionProxy submissionProxy, string memory json) internal {
        if (!vm.keyExists(json, ".deployFeed")) {
            return;
        }
        bytes memory raw = json.parseRaw(".deployFeed.feedNames");
        string[] memory feedNames = abi.decode(raw, (string[]));

        bytes32[] memory feedHashes = new bytes32[](feedNames.length);
        address[] memory feedAddresses = new address[](feedNames.length);
        address[] memory proxyAddresses = new address[](feedNames.length);
        for (uint256 j = 0; j < feedNames.length; j++) {
            Feed feed = new Feed(DECIMALS, feedNames[j], address(submissionProxy));
            console.log("(Feed Deployed)", feedNames[j], address(feed));
            FeedProxy feedProxy = new FeedProxy(address(feed));
            console.log("(FeedProxy Deployed)", feedNames[j], address(feedProxy));

            config.storeFeedAddress(feedNames[j], address(feed), address(feedProxy));

            feedHashes[j] = string2bytes32Hash(feedNames[j]);
            proxyAddresses[j] = address(feedProxy);
            feedAddresses[j] = address(feed);
            console.log("(Feed Prepared for updateFeed)", feedNames[j], address(feed));
        }
        submissionProxy.updateFeedBulk(feedHashes, feedAddresses);
        console.log("(Feeds Updated)");

        if (!vm.keyExists(json, ".feedRouter")) {
            return;
        }

        FeedRouter feedRouter;
        if (vm.keyExists(json, ".feedRouter.address")) {
            feedRouter = useExistingFeedRouter(json);
            console.log("(Use existing FeedRouter)", address(feedRouter));
        }else {
            feedRouter = new FeedRouter();
            console.log("(FeedRouter Deployed)", address(feedRouter));
        }

        config.storeAddress("FeedRouter", address(feedRouter));
        feedRouter.updateProxyBulk(feedNames, proxyAddresses);
        console.log("(Proxies Updated)");
    }

    function string2bytes32Hash(string memory str) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(str));
    }
}
