// SPDX-License-Identifier: MIT
pragma solidity 0.8.20;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IAggregator} from "./interfaces/IAggregatorSubmit.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

contract SubmissionProxy is Ownable {
    using ECDSA for bytes32;
    uint256 maxSubmission = 50;
    address[] public oracleAddresses;
    mapping(address => bool) oracles;

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();

    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);

    constructor() Ownable(msg.sender) {}

    function getOracles() public view returns (address[] memory) {
        return oracleAddresses;
    }

    function addOracle(address _oracle) public onlyOwner {
        oracleAddresses.push(_oracle);
        oracles[_oracle] = true;
        emit OracleAdded(_oracle);
    }

    function removeOracle(address _oracle) public onlyOwner {
        if (!oracles[_oracle]) {
            revert InvalidOracle();
        }
        for (uint256 i = 0; i < oracleAddresses.length; ++i) {
            if (oracleAddresses[i] == _oracle) {
                address last = oracleAddresses[oracleAddresses.length - 1];
                oracleAddresses[i] = last;
                oracleAddresses.pop();
                delete oracles[_oracle];
                emit OracleRemoved(_oracle);
                break;
            }
        }
    }

    function setMaxSubmission(uint256 _maxSubmission) public onlyOwner {
        maxSubmission = _maxSubmission;
    }

    modifier onlyOracle() {
        if (!oracles[msg.sender]) revert OnlyOracle();
        _;
    }

    function batchSubmit(
        address[] memory _aggregators,
        int256[] memory _submissions,
        bytes memory _signature
    ) public {
        bytes32 hashData = keccak256(abi.encodePacked(_aggregators, _submissions));
        verify(hashData, _signature);
        if (_aggregators.length != _submissions.length) revert InvalidSubmissionLength();
        for (uint256 i = 0; i < _aggregators.length; i++) {
            IAggregator(_aggregators[i]).submit(_submissions[i]);
        }
    }

    function verify(bytes32 hashData, bytes memory signature) internal view {
        address oracleAddress = MessageHashUtils.toEthSignedMessageHash(hashData).recover(
            signature
        );
        if (!oracles[oracleAddress]) revert OnlyOracle();
    }
}
