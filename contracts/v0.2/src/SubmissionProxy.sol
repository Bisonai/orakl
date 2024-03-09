// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IAggregator} from "./interfaces/IAggregatorSubmit.sol";

contract SubmissionProxy is Ownable {
    uint256 maxSubmission = 50;
    address[] public oracleAddresses;
    mapping(address => bool) oracles;

    error OnlyOracle();
    error InvalidOracle();
    error InvalidSubmissionLength();

    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);
    event MaxSubmissionSet(uint256 maxSubmission);

    modifier onlyOracle() {
        if (!oracles[msg.sender]) revert OnlyOracle();
        _;
    }

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

	uint256 oracleAddressesLength = oracleAddresses.length;
        for (uint256 i = 0; i < oracleAddressesLength; ++i) {
            if (oracleAddresses[i] == _oracle) {
                address last = oracleAddresses[oracleAddressesLength - 1];
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

    function submit(address[] memory _aggregators, int256[] memory _submissions) public onlyOracle {
        if (_aggregators.length != _submissions.length) revert InvalidSubmissionLength();
        for (uint256 i = 0; i < _aggregators.length; ++i) {
            IAggregator(_aggregators[i]).submit(_submissions[i]);
        }
    }
}
