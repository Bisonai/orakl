// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2, console} from "forge-std/Test.sol";
import {SubmissionProxy} from "../src/SubmissionProxy.sol";
import {Aggregator} from "../src/Aggregator.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

contract SubmissionProxyTest is Test {
    SubmissionProxy batchSubmission;
    uint32 timeout = 10;
    address validator = address(0);
    uint8 decimals = 18;
    string description = "Test Aggregator";
    uint256 timestamp = 1706170779;
    uint256 internal userPrivateKey;
    uint256 internal signerPrivateKey;

    function estimateGasCost(uint256 startGas) internal view returns (uint256) {
        return startGas - gasleft();
    }

    function setUp() public {
        vm.warp(timestamp);
        batchSubmission = new SubmissionProxy();
        userPrivateKey = 0xa11ce;
        signerPrivateKey = 0xabc123;
    }

    function test_AddAndRemoveOracle() public {
        batchSubmission.addOracle(address(0));
        uint256 oracleLength = batchSubmission.getOracles().length;
        assertEq(oracleLength, 1);

        batchSubmission.removeOracle(address(0));
        oracleLength = batchSubmission.getOracles().length;
        assertEq(oracleLength, 0);
    }

    function test_BatchSubmission() public {
        address user = vm.addr(userPrivateKey);
        address signer = vm.addr(signerPrivateKey);
        batchSubmission.addOracle(address(0));
        batchSubmission.addOracle(signer);
        address[] memory oracleRemove;
        address[] memory oracleAdd = new address[](2);
        uint256 submitGas;
        uint256 batchSubmitGas;
        address[] memory aggregatorForBathSubmit = new address[](50);
        int256[] memory submissions = new int256[](50);
        uint256 startGas;

        for (uint i = 0; i < 50; i++) {
            Aggregator aggregator = new Aggregator(timeout, decimals, description);
            oracleAdd[0] = address(batchSubmission);
            oracleAdd[1] = address(0);
            aggregator.changeOracles(oracleRemove, oracleAdd, 1, 1, 0);
            aggregatorForBathSubmit[i] = address(aggregator);
            submissions[i] = 10;
            startGas = gasleft();
            vm.prank(address(0));
            aggregator.submit(10);
            submitGas += estimateGasCost(startGas);
        }
        startGas = gasleft();
        vm.startPrank(signer);
        bytes32 digest = MessageHashUtils.toEthSignedMessageHash(
            keccak256(abi.encodePacked(aggregatorForBathSubmit, submissions))
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(signerPrivateKey, digest);
        bytes memory signature = abi.encodePacked(r, s, v);
        vm.stopPrank();
        vm.prank(user);
        batchSubmission.batchSubmit(aggregatorForBathSubmit, submissions, signature);
        batchSubmitGas = estimateGasCost(startGas);

        console.log("submit", submitGas, "batch submit", batchSubmitGas);
    }

    function test_RevertedWithOnlyOracle() public {
        address user = vm.addr(userPrivateKey);
        address signer = vm.addr(signerPrivateKey);
        Aggregator aggregator = new Aggregator(timeout, decimals, description);
        address[] memory oracleRemove;
        address[] memory oracleAdd = new address[](1);
        oracleAdd[0] = address(batchSubmission);

        aggregator.changeOracles(oracleRemove, oracleAdd, 1, 1, 0);
        address[] memory aggregatorForBathSubmit = new address[](1);
        int256[] memory submissions = new int256[](1);

        aggregatorForBathSubmit[0] = address(aggregator);
        submissions[0] = 1;

        vm.startPrank(signer);
        bytes32 digest = MessageHashUtils.toEthSignedMessageHash(
            keccak256(abi.encodePacked(aggregatorForBathSubmit, submissions))
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(signerPrivateKey, digest);
        bytes memory signature = abi.encodePacked(r, s, v);
        vm.stopPrank();

        vm.expectRevert(SubmissionProxy.OnlyOracle.selector);
        vm.prank(user);
        batchSubmission.batchSubmit(aggregatorForBathSubmit, submissions, signature);
    }
}
