// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IRegistry.sol";
import "./L1EndpointBase.sol";
import "./L1EndpointRequestResponse.sol";
import "./L1EndpointVRF.sol";

contract L1Endpoint is Ownable, L1EndpointBase, L1EndpointVRF, L1EndpointRequestResponse {
    error FailedToDeposit();

    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);

    constructor(
        address registryAddress,
        address vrfCoordinator,
        address requestResponseCoordinator
    )
        L1EndpointBase(registryAddress)
        L1EndpointVRF(vrfCoordinator)
        L1EndpointRequestResponse(requestResponseCoordinator)
    {}

    receive() external payable {}

    function addOracle(address oracle) public onlyOwner {
        sOracles[oracle] = true;
        emit OracleAdded(oracle);
    }

    function removeOracle(address oracle) public onlyOwner {
        delete sOracles[oracle];
        emit OracleRemoved(oracle);
    }
}
