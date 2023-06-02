// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface ICoordinatorBase {
    // Fee configuration that separates fees based on the number of
    // requests created per account. This applies only to [regular]
    // account.
    struct FeeConfig {
        // Flat fee charged per fulfillment in millionths of KLAY
        // So fee range is [0, 2^32/10^6].
        uint32 fulfillmentFlatFeeKlayPPMTier1;
        uint32 fulfillmentFlatFeeKlayPPMTier2;
        uint32 fulfillmentFlatFeeKlayPPMTier3;
        uint32 fulfillmentFlatFeeKlayPPMTier4;
        uint32 fulfillmentFlatFeeKlayPPMTier5;
        uint24 reqsForTier2;
        uint24 reqsForTier3;
        uint24 reqsForTier4;
        uint24 reqsForTier5;
    }

    /**
     * @notice Sets the configuration of the VRF coordinator
     * @param maxGasLimit global max for request gas limit
     * @param gasAfterPaymentCalculation gas used in doing accounting
     * after completing the gas measurement
     * @param feeConfig fee tier configuration
     */
    function setConfig(
        uint32 maxGasLimit,
        uint32 gasAfterPaymentCalculation,
        FeeConfig memory feeConfig
    ) external;

    /**
     * @notice Check to see if there exists a request commitment
     * consumers for all consumers and keyhashes for a given acc.
     * @param accId - ID of the account
     * @return true if there exists at least one unfulfilled request
     * for the account, false otherwise.
     */
    function pendingRequestExists(
        address consumer,
        uint64 accId,
        uint64 nonce
    ) external view returns (bool);

    /**
     * @notice Get request commitment.
     * @param requestId id of request
     * @return commmitment value that can be used to determine whether
     * a request is fulfilled or not. If `requestId` is valid and
     * commitment equals to bytes32(0), the request was fulfilled.
     */
    function getCommitment(uint256 requestId) external view returns (bytes32);

    /**
     * @notice Canceling oracle request
     * @param requestId - ID of the Oracle Request
     */
    function cancelRequest(uint256 requestId) external;

    /**
     * @notice Access address for prepayment associated with
     * @notice coordinator.
     * @return prepayment address
     */
    function getPrepaymentAddress() external returns (address);
}
