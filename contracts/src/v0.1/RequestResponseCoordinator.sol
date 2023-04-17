// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/ICoordinatorBase.sol";
import "./interfaces/IRequestResponseCoordinator.sol";
import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";
import "./RequestResponseConsumerBase.sol";
import "./RequestResponseConsumerFulfill.sol";
import "./libraries/Orakl.sol";

contract RequestResponseCoordinator is
    Ownable,
    ICoordinatorBase,
    IRequestResponseCoordinator,
    ITypeAndVersion
{
    using Orakl for Orakl.Request;

    // 5k is plenty for an EXTCODESIZE call (2600) + warm CALL (100)
    // and some arithmetic operations.
    uint256 private constant GAS_FOR_CALL_EXACT_CHECK = 5_000;

    address[] public sOracles;
    mapping(address => bool) private sIsOracleRegistered;

    /* requestID */
    /* commitment */
    mapping(uint256 => bytes32) private sRequestIdToCommitment;

    /* requestID */
    /* owner */
    mapping(uint256 => address) private sRequestOwner;

    uint256 public sMinBalance;

    IPrepayment private sPrepayment;

    struct Config {
        uint32 maxGasLimit;
        // Reentrancy protection.
        bool reentrancyLock;
        // Gas to cover oracle payment after we calculate the payment.
        // We make it configurable in case those operations are repriced.
        uint32 gasAfterPaymentCalculation;
    }
    Config private sConfig;

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
    FeeConfig private sFeeConfig;

    struct DirectPaymentConfig {
        uint256 fulfillmentFee;
        uint256 baseFee;
    }

    DirectPaymentConfig private sDirectPaymentConfig;
    mapping(bytes32 => bool) sJobId;

    error InvalidConsumer(uint64 accId, address consumer);
    error InvalidAccount();
    error UnregisteredOracleFulfillment(address oracle);
    error NoCorrespondingRequest();
    error IncorrectCommitment();
    error NotRequestOwner();
    error Reentrant();
    error InsufficientPayment(uint256 have, uint256 want);
    error RefundFailure();
    error GasLimitTooBig(uint32 have, uint32 want);
    error OracleAlreadyRegistered(address oracle);
    error NoSuchOracle(address oracle);
    error InvalidJobId();

    event DataRequested(
        uint256 indexed requestId,
        bytes32 jobId,
        uint64 indexed accId,
        uint32 callbackGasLimit,
        address indexed sender,
        bool isDirectPayment,
        bytes data
    );

    event DataRequestCancelled(uint256 indexed requestId);
    event ConfigSet(uint32 maxGasLimit, uint32 gasAfterPaymentCalculation, FeeConfig feeConfig);
    event DirectPaymentConfigSet(uint256 fulfillmentFee, uint256 baseFee);

    event OracleRegistered(address oracle);
    event OracleDeregistered(address oracle);
    event MinBalanceSet(uint256 minBalance);
    event PrepaymentSet(address prepayment);

    event DataRequestFulfilledUint256(
        uint256 indexed requestId,
        uint256 response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledInt256(
        uint256 indexed requestId,
        int256 response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledBool(
        uint256 indexed requestId,
        bool response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledString(
        uint256 indexed requestId,
        string response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledBytes32(
        uint256 indexed requestId,
        bytes32 response,
        uint256 payment,
        bool success
    );
    event DataRequestFulfilledBytes(
        uint256 indexed requestId,
        bytes response,
        uint256 payment,
        bool success
    );

    modifier nonReentrant() {
        if (sConfig.reentrancyLock) {
            revert Reentrant();
        }
        _;
    }

    constructor(address prepayment) {
        sJobId[keccak256(abi.encodePacked("uint256"))] = true;
        sJobId[keccak256(abi.encodePacked("int256"))] = true;
        sJobId[keccak256(abi.encodePacked("bool"))] = true;
        sJobId[keccak256(abi.encodePacked("string"))] = true;
        sJobId[keccak256(abi.encodePacked("bytes32"))] = true;
        sJobId[keccak256(abi.encodePacked("bytes"))] = true;

        sPrepayment = IPrepayment(prepayment);
        emit PrepaymentSet(prepayment);
    }

    /**
     * @notice Register an oracle
     * @param oracle address of the oracle
     */
    function registerOracle(address oracle) external onlyOwner {
        if (sIsOracleRegistered[oracle]) {
            revert OracleAlreadyRegistered(oracle);
        }
        sOracles.push(oracle);
        sIsOracleRegistered[oracle] = true;
        emit OracleRegistered(oracle);
    }

    /**
     * @notice Deregister an oracle
     * @param oracle address of the oracle
     */
    function deregisterOracle(address oracle) external onlyOwner {
        if (!sIsOracleRegistered[oracle]) {
            revert NoSuchOracle(oracle);
        }
        delete sIsOracleRegistered[oracle];

        uint256 oraclesLength = sOracles.length;
        for (uint256 i; i < oraclesLength; ++i) {
            if (sOracles[i] == oracle) {
                address last = sOracles[oraclesLength - 1];
                sOracles[i] = last;
                sOracles.pop();
                break;
            }
        }

        emit OracleDeregistered(oracle);
    }

    /**
     * @notice Sets the general configuration
     * @param maxGasLimit global max for request gas limit
     * @param gasAfterPaymentCalculation gas used in doing accounting after completing the gas measurement
     * @param feeConfig fee tier configuration
     */
    function setConfig(
        uint32 maxGasLimit,
        uint32 gasAfterPaymentCalculation,
        FeeConfig memory feeConfig
    ) external onlyOwner {
        sConfig = Config({
            maxGasLimit: maxGasLimit,
            gasAfterPaymentCalculation: gasAfterPaymentCalculation,
            reentrancyLock: false
        });
        sFeeConfig = feeConfig;
        emit ConfigSet(maxGasLimit, gasAfterPaymentCalculation, sFeeConfig);
    }

    function getConfig()
        external
        view
        returns (uint32 maxGasLimit, uint32 gasAfterPaymentCalculation)
    {
        return (sConfig.maxGasLimit, sConfig.gasAfterPaymentCalculation);
    }

    function getFeeConfig()
        external
        view
        returns (
            uint32 fulfillmentFlatFeeKlayPPMTier1,
            uint32 fulfillmentFlatFeeKlayPPMTier2,
            uint32 fulfillmentFlatFeeKlayPPMTier3,
            uint32 fulfillmentFlatFeeKlayPPMTier4,
            uint32 fulfillmentFlatFeeKlayPPMTier5,
            uint24 reqsForTier2,
            uint24 reqsForTier3,
            uint24 reqsForTier4,
            uint24 reqsForTier5
        )
    {
        return (
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier1,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier2,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier3,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier4,
            sFeeConfig.fulfillmentFlatFeeKlayPPMTier5,
            sFeeConfig.reqsForTier2,
            sFeeConfig.reqsForTier3,
            sFeeConfig.reqsForTier4,
            sFeeConfig.reqsForTier5
        );
    }

    function setDirectPaymentConfig(
        DirectPaymentConfig memory directPaymentConfig
    ) external onlyOwner {
        sDirectPaymentConfig = directPaymentConfig;
        emit DirectPaymentConfigSet(
            directPaymentConfig.fulfillmentFee,
            directPaymentConfig.baseFee
        );
    }

    function getDirectPaymentConfig() external view returns (uint256, uint256) {
        return (sDirectPaymentConfig.fulfillmentFee, sDirectPaymentConfig.baseFee);
    }

    function getPrepaymentAddress() external view returns (address) {
        return address(sPrepayment);
    }

    function setMinBalance(uint256 minBalance) external onlyOwner {
        sMinBalance = minBalance;
        emit MinBalanceSet(minBalance);
    }

    // TODO description
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint64 accId
    ) external nonReentrant returns (uint256) {
        // TODO check if he is one of the consumers

        if (!sJobId[req.id]) {
            revert InvalidJobId();
        }

        uint256 balance = sPrepayment.getBalance(accId);
        if (balance < sMinBalance) {
            revert InsufficientPayment(balance, sMinBalance);
        }

        bool isDirectPayment = false;
        uint256 requestId = requestDataInternal(req, accId, callbackGasLimit, isDirectPayment);

        return requestId;
    }

    // TODO description
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit
    ) external payable returns (uint256) {
        uint256 fee = estimateDirectPaymentFee();
        if (msg.value < fee) {
            revert InsufficientPayment(msg.value, fee);
        }

        uint64 accId = sPrepayment.createTemporaryAccount();
        bool isDirectPayment = true;
        uint256 requestId = requestDataInternal(req, accId, callbackGasLimit, isDirectPayment);
        sPrepayment.depositTemporary{value: fee}(accId);

        uint256 remaining = msg.value - fee;
        if (remaining > 0) {
            (bool sent, ) = msg.sender.call{value: remaining}("");
            if (!sent) {
                revert RefundFailure();
            }
        }

        return requestId;
    }

    /**
     * @inheritdoc IRequestResponseCoordinator
     */
    function cancelRequest(uint256 requestId) external {
        bytes32 commitment = sRequestIdToCommitment[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }

        if (sRequestOwner[requestId] != msg.sender) {
            revert NotRequestOwner();
        }

        delete sRequestIdToCommitment[requestId];
        delete sRequestOwner[requestId];

        emit DataRequestCancelled(requestId);
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "RequestResponseCoordinator v0.1";
    }

    /**
     * @notice Find out whether given oracle address was registered.
     * @return true when oracle address registered, otherwise false
     */
    function isOracleRegistered(address oracle) external view returns (bool) {
        return sIsOracleRegistered[oracle];
    }

    function estimateDirectPaymentFee() internal view returns (uint256) {
        return sDirectPaymentConfig.fulfillmentFee + sDirectPaymentConfig.baseFee;
    }

    function requestDataInternal(
        Orakl.Request memory req,
        uint64 accId,
        uint32 callbackGasLimit,
        bool isDirectPayment
    ) internal returns (uint256) {
        sPrepayment.isValidAccount(accId);

        // Its important to ensure that the consumer is in fact who they say they
        // are, otherwise they could use someone else's account balance.
        // A nonce of 0 indicates consumer is not allocated to the acc.
        uint64 currentNonce = sPrepayment.getNonce(accId, msg.sender);
        if (currentNonce == 0) {
            revert InvalidConsumer(accId, msg.sender);
        }

        // TODO update comment
        // No lower bound on the requested gas limit. A user could request 0
        // and they would simply be billed for the proof verification and wouldn't be
        // able to do anything with the random value.
        if (callbackGasLimit > sConfig.maxGasLimit) {
            revert GasLimitTooBig(callbackGasLimit, sConfig.maxGasLimit);
        }

        uint64 nonce = sPrepayment.increaseNonce(accId, msg.sender);

        uint256 requestId = computeRequestId(msg.sender, accId, nonce);
        sRequestIdToCommitment[requestId] = keccak256(
            abi.encode(requestId, block.number, accId, callbackGasLimit, msg.sender)
        );

        sRequestOwner[requestId] = msg.sender;

        emit DataRequested(
            requestId,
            req.id,
            accId,
            callbackGasLimit,
            msg.sender,
            isDirectPayment,
            req.buf.buf
        );

        return requestId;
    }

    function calculatePaymentAmount(
        uint256 startGas,
        uint256 gasAfterPaymentCalculation,
        uint32 fulfillmentFlatFeeKlayPPM
    ) internal view returns (uint256) {
        uint256 paymentNoFee = tx.gasprice * (gasAfterPaymentCalculation + startGas - gasleft());
        uint256 fee = 1e12 * uint256(fulfillmentFlatFeeKlayPPM);
        return paymentNoFee + fee;
    }

    /**
     * @inheritdoc ICoordinatorBase
     */
    function pendingRequestExists(
        address consumer,
        uint64 accId,
        uint64 nonce
    ) public view returns (bool) {
        uint256 oraclesLength = sOracles.length;
        for (uint256 i; i < oraclesLength; ++i) {
            uint256 reqId = computeRequestId(consumer, accId, nonce);
            if (sRequestIdToCommitment[reqId] != 0) {
                return true;
            }
        }
        return false;
    }

    /*
     * @notice Compute fee based on the request count
     * @param reqCount number of requests
     * @return feePPM fee in KLAY PPM
     */
    function getFeeTier(uint64 reqCount) public view returns (uint32) {
        FeeConfig memory fc = sFeeConfig;
        if (0 <= reqCount && reqCount <= fc.reqsForTier2) {
            return fc.fulfillmentFlatFeeKlayPPMTier1;
        }
        if (fc.reqsForTier2 < reqCount && reqCount <= fc.reqsForTier3) {
            return fc.fulfillmentFlatFeeKlayPPMTier2;
        }
        if (fc.reqsForTier3 < reqCount && reqCount <= fc.reqsForTier4) {
            return fc.fulfillmentFlatFeeKlayPPMTier3;
        }
        if (fc.reqsForTier4 < reqCount && reqCount <= fc.reqsForTier5) {
            return fc.fulfillmentFlatFeeKlayPPMTier4;
        }
        return fc.fulfillmentFlatFeeKlayPPMTier5;
    }

    function computeRequestId(
        address sender,
        uint64 accId,
        uint64 nonce
    ) private pure returns (uint256) {
        return uint256(keccak256(abi.encode(sender, accId, nonce)));
    }

    /**
     * @dev calls target address with exactly gasAmount gas and data as calldata
     * or reverts if at least gasAmount gas is not available.
     */
    function callWithExactGas(
        uint256 gasAmount,
        address target,
        bytes memory data
    ) private returns (bool success) {
        // solhint-disable-next-line no-inline-assembly
        assembly {
            let g := gas()
            // Compute g -= GAS_FOR_CALL_EXACT_CHECK and check for underflow
            // The gas actually passed to the callee is min(gasAmount, 63//64*gas available).
            // We want to ensure that we revert if gasAmount >  63//64*gas available
            // as we do not want to provide them with less, however that check itself costs
            // gas.  GAS_FOR_CALL_EXACT_CHECK ensures we have at least enough gas to be able
            // to revert if gasAmount >  63//64*gas available.
            if lt(g, GAS_FOR_CALL_EXACT_CHECK) {
                revert(0, 0)
            }
            g := sub(g, GAS_FOR_CALL_EXACT_CHECK)
            // if g - g//64 <= gasAmount, revert
            // (we subtract g//64 because of EIP-150)
            if iszero(gt(sub(g, div(g, 64)), gasAmount)) {
                revert(0, 0)
            }
            // solidity calls check that a contract actually exists at the destination, so we do the same
            if iszero(extcodesize(target)) {
                revert(0, 0)
            }
            // call and return whether we succeeded. ignore return data
            // call(gas,addr,value,argsOffset,argsLength,retOffset,retLength)
            success := call(gasAmount, target, 0, add(data, 0x20), mload(data), 0, 0)
        }
        return success;
    }

    function validateDataResponse(
        RequestCommitment memory rc,
        uint256 requestId
    ) internal returns (bool) {
        if (!sIsOracleRegistered[msg.sender]) {
            revert UnregisteredOracleFulfillment(msg.sender);
        }

        bytes32 commitment = sRequestIdToCommitment[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }

        if (
            commitment !=
            keccak256(abi.encode(requestId, rc.blockNum, rc.accId, rc.callbackGasLimit, rc.sender))
        ) {
            revert IncorrectCommitment();
        }

        delete sRequestIdToCommitment[requestId];

        return true;
    }

    function fulfill(bytes memory resp, RequestCommitment memory rc) internal returns (bool) {
        // Call with explicitly the amount of callback gas requested
        // Important to not let them exhaust the gas budget and avoid oracle payment.
        // Do not allow any non-view/non-pure coordinator functions to be called
        // during the consumers callback code via reentrancyLock.
        // Note that callWithExactGas will revert if we do not have sufficient gas
        // to give the callee their requested amount.
        sConfig.reentrancyLock = true;
        bool success = callWithExactGas(rc.callbackGasLimit, rc.sender, resp);
        sConfig.reentrancyLock = false;
        return success;
    }

    function pay(
        RequestCommitment memory rc,
        bool isDirectPayment,
        uint256 startGas
    ) internal returns (uint256 payment) {
        if (isDirectPayment) {
            payment = sPrepayment.chargeFeeTemporary(rc.accId, msg.sender);
            sPrepayment.increaseReqCountTemporary(rc.accId);
        } else {
            uint64 reqCount = sPrepayment.getReqCount(rc.accId);
            payment = calculatePaymentAmount(
                startGas,
                sConfig.gasAfterPaymentCalculation,
                getFeeTier(reqCount)
            );
            sPrepayment.chargeFee(rc.accId, payment, msg.sender);
            sPrepayment.increaseReqCount(rc.accId);
        }
    }

    function fulfillDataRequestUint256(
        uint256 requestId,
        uint256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);
        RequestResponseConsumerFulfillUint256 rr;
        bytes memory resp = abi.encodeWithSelector(
            rr.rawFulfillDataRequestUint256.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas);
        emit DataRequestFulfilledUint256(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestInt256(
        uint256 requestId,
        int256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);
        RequestResponseConsumerFulfillInt256 rr;
        bytes memory resp = abi.encodeWithSelector(
            rr.rawFulfillDataRequestInt256.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas);
        emit DataRequestFulfilledInt256(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestBool(
        uint256 requestId,
        bool response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);
        RequestResponseConsumerFulfillBool rr;
        bytes memory resp = abi.encodeWithSelector(
            rr.rawFulfillDataRequestBool.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas);
        emit DataRequestFulfilledBool(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestString(
        uint256 requestId,
        string memory response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);
        RequestResponseConsumerFulfillString rr;
        bytes memory resp = abi.encodeWithSelector(
            rr.rawFulfillDataRequestString.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas);
        emit DataRequestFulfilledString(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestBytes32(
        uint256 requestId,
        bytes32 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);
        RequestResponseConsumerFulfillBytes32 rr;
        bytes memory resp = abi.encodeWithSelector(
            rr.rawFulfillDataRequestBytes32.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas);
        emit DataRequestFulfilledBytes32(requestId, response, payment, success);
        return payment;
    }

    function fulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);
        RequestResponseConsumerFulfillBytes rr;
        bytes memory resp = abi.encodeWithSelector(
            rr.rawFulfillDataRequestBytes.selector,
            requestId,
            response
        );
        bool success = fulfill(resp, rc);
        uint256 payment = pay(rc, isDirectPayment, startGas);
        emit DataRequestFulfilledBytes(requestId, response, payment, success);
        return payment;
    }
}
