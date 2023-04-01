// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/CoordinatorBaseInterface.sol";
import "./interfaces/RequestResponseCoordinatorInterface.sol";
import "./interfaces/PrepaymentInterface.sol";
import "./interfaces/TypeAndVersionInterface.sol";
import "./RequestResponseConsumerBase.sol";
import "./libraries/Orakl.sol";

contract RequestResponseCoordinator is
    CoordinatorBaseInterface,
    Ownable,
    RequestResponseCoordinatorInterface,
    TypeAndVersionInterface
{
    using Orakl for Orakl.Request;

    // 5k is plenty for an EXTCODESIZE call (2600) + warm CALL (100)
    // and some arithmetic operations.
    uint256 private constant GAS_FOR_CALL_EXACT_CHECK = 5_000;

    /* oracle address */
    /* registration status */
    mapping(address => bool) private s_oracles;

    /* requestID */
    /* commitment */
    mapping(uint256 => bytes32) private s_requestCommitments;

    /* requestID */
    /* owner */
    mapping(uint256 => address) private s_requestOwner;

    address[] private s_registeredOracles;

    uint256 public s_minBalance;

    PrepaymentInterface s_prepayment;

    mapping(bytes4 => uint256) private sFulfillFunctionSelector;

    struct Config {
        uint32 maxGasLimit;
        // Reentrancy protection.
        bool reentrancyLock;
        // Gas to cover oracle payment after we calculate the payment.
        // We make it configurable in case those operations are repriced.
        uint32 gasAfterPaymentCalculation;
    }
    Config private s_config;

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
    FeeConfig private s_feeConfig;

    struct DirectPaymentConfig {
        uint256 fulfillmentFee;
        uint256 baseFee;
    }

    DirectPaymentConfig s_directPaymentConfig;

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
        if (s_config.reentrancyLock) {
            revert Reentrant();
        }
        _;
    }

    constructor(address prepayment) {
        s_prepayment = PrepaymentInterface(prepayment);
        emit PrepaymentSet(prepayment);
    }

    /**
     * @notice Register an oracle
     * @param oracle address of the oracle
     */
    function registerOracle(address oracle) external onlyOwner {
        if (s_oracles[oracle]) {
            revert OracleAlreadyRegistered(oracle);
        }
        s_oracles[oracle] = true;
        s_registeredOracles.push(oracle);
        emit OracleRegistered(oracle);
    }

    /**
     * @notice Deregister an oracle
     * @param oracle address of the oracle
     */
    function deregisterOracle(address oracle) external onlyOwner {
        if (!s_oracles[oracle]) {
            revert NoSuchOracle(oracle);
        }
        delete s_oracles[oracle];
        for (uint256 i = 0; i < s_registeredOracles.length; i++) {
            if (s_registeredOracles[i] == oracle) {
                address last = s_registeredOracles[s_registeredOracles.length - 1];
                s_registeredOracles[i] = last;
                s_registeredOracles.pop();
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
        s_config = Config({
            maxGasLimit: maxGasLimit,
            gasAfterPaymentCalculation: gasAfterPaymentCalculation,
            reentrancyLock: false
        });
        s_feeConfig = feeConfig;
        emit ConfigSet(maxGasLimit, gasAfterPaymentCalculation, s_feeConfig);
    }

    function getConfig()
        external
        view
        returns (uint32 maxGasLimit, uint32 gasAfterPaymentCalculation)
    {
        return (s_config.maxGasLimit, s_config.gasAfterPaymentCalculation);
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
            s_feeConfig.fulfillmentFlatFeeKlayPPMTier1,
            s_feeConfig.fulfillmentFlatFeeKlayPPMTier2,
            s_feeConfig.fulfillmentFlatFeeKlayPPMTier3,
            s_feeConfig.fulfillmentFlatFeeKlayPPMTier4,
            s_feeConfig.fulfillmentFlatFeeKlayPPMTier5,
            s_feeConfig.reqsForTier2,
            s_feeConfig.reqsForTier3,
            s_feeConfig.reqsForTier4,
            s_feeConfig.reqsForTier5
        );
    }

    function setDirectPaymentConfig(
        DirectPaymentConfig memory directPaymentConfig
    ) public onlyOwner {
        s_directPaymentConfig = directPaymentConfig;
        emit DirectPaymentConfigSet(
            directPaymentConfig.fulfillmentFee,
            directPaymentConfig.baseFee
        );
    }

    function getDirectPaymentConfig() external view returns (uint256, uint256) {
        return (s_directPaymentConfig.fulfillmentFee, s_directPaymentConfig.baseFee);
    }

    function estimateDirectPaymentFee() public view returns (uint256) {
        return s_directPaymentConfig.fulfillmentFee + s_directPaymentConfig.baseFee;
    }

    function getPrepaymentAddress() public view returns (address) {
        return address(s_prepayment);
    }

    function setMinBalance(uint256 minBalance) public onlyOwner {
        s_minBalance = minBalance;
        emit MinBalanceSet(minBalance);
    }

    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint64 accId
    ) external nonReentrant returns (uint256 requestId) {
        bool isDirectPayment = false;
        (uint256 balance, , , ) = s_prepayment.getAccount(accId);
        if (balance < s_minBalance) {
            revert InsufficientPayment(balance, s_minBalance);
        }
        requestId = requestDataInternal(req, accId, callbackGasLimit, isDirectPayment);
    }

    function requestDataInternal(
        Orakl.Request memory req,
        uint64 accId,
        uint32 callbackGasLimit,
        bool isDirectPayment
    ) internal returns (uint256) {
        // Input validation using the account storage.
        // call to prepayment contract
        address owner = s_prepayment.getAccountOwner(accId);
        if (owner == address(0)) {
            revert InvalidAccount();
        }

        // Its important to ensure that the consumer is in fact who they say they
        // are, otherwise they could use someone else's account balance.
        // A nonce of 0 indicates consumer is not allocated to the acc.
        uint64 currentNonce = s_prepayment.getNonce(msg.sender, accId);
        if (currentNonce == 0) {
            revert InvalidConsumer(accId, msg.sender);
        }

        // TODO update comment
        // No lower bound on the requested gas limit. A user could request 0
        // and they would simply be billed for the proof verification and wouldn't be
        // able to do anything with the random value.
        if (callbackGasLimit > s_config.maxGasLimit) {
            revert GasLimitTooBig(callbackGasLimit, s_config.maxGasLimit);
        }

        uint64 nonce = s_prepayment.increaseNonce(msg.sender, accId);

        uint256 requestId = computeRequestId(msg.sender, accId, nonce);
        s_requestCommitments[requestId] = keccak256(
            abi.encode(requestId, block.number, accId, callbackGasLimit, msg.sender)
        );

        s_requestOwner[requestId] = msg.sender;

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

    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit
    ) external payable returns (uint256) {
        uint256 fee = estimateDirectPaymentFee();
        if (msg.value < fee) {
            revert InsufficientPayment(msg.value, fee);
        }

        uint64 accId = s_prepayment.createAccount();
        s_prepayment.addConsumer(accId, msg.sender);
        bool isDirectPayment = true;
        uint256 requestId = requestDataInternal(req, accId, callbackGasLimit, isDirectPayment);
        s_prepayment.deposit{value: fee}(accId);

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
     * @inheritdoc CoordinatorBaseInterface
     */
    function pendingRequestExists(
        address consumer,
        uint64 accId,
        uint64 nonce
    ) public view returns (bool) {
        for (uint256 i = 0; i < s_registeredOracles.length; i++) {
            uint256 reqId = computeRequestId(consumer, accId, nonce);
            if (s_requestCommitments[reqId] != 0) {
                return true;
            }
        }
        return false;
    }

    /**
     * @inheritdoc RequestResponseCoordinatorInterface
     */
    function cancelRequest(uint256 requestId) external {
        bytes32 commitment = s_requestCommitments[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }

        if (s_requestOwner[requestId] != msg.sender) {
            revert NotRequestOwner();
        }

        delete s_requestCommitments[requestId];
        delete s_requestOwner[requestId];

        emit DataRequestCancelled(requestId);
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "RequestResponseCoordinator v0.1";
    }

    /*
     * @notice Compute fee based on the request count
     * @param reqCount number of requests
     * @return feePPM fee in KLAY PPM
     */
    function getFeeTier(uint64 reqCount) public view returns (uint32) {
        FeeConfig memory fc = s_feeConfig;
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

    function calculatePaymentAmount(
        uint256 startGas,
        uint256 gasAfterPaymentCalculation,
        uint32 fulfillmentFlatFeeKlayPPM
    ) internal view returns (uint256) {
        uint256 paymentNoFee = tx.gasprice * (gasAfterPaymentCalculation + startGas - gasleft());
        uint256 fee = 1e12 * uint256(fulfillmentFlatFeeKlayPPM);
        return paymentNoFee + fee;
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
        if (!s_oracles[msg.sender]) {
            revert UnregisteredOracleFulfillment(msg.sender);
        }

        bytes32 commitment = s_requestCommitments[requestId];
        if (commitment == 0) {
            revert NoCorrespondingRequest();
        }

        if (
            commitment !=
            keccak256(abi.encode(requestId, rc.blockNum, rc.accId, rc.callbackGasLimit, rc.sender))
        ) {
            revert IncorrectCommitment();
        }

        delete s_requestCommitments[requestId];

        return true;
    }

    function fulfill(bytes memory resp, RequestCommitment memory rc) internal returns (bool) {
        // Call with explicitly the amount of callback gas requested
        // Important to not let them exhaust the gas budget and avoid oracle payment.
        // Do not allow any non-view/non-pure coordinator functions to be called
        // during the consumers callback code via reentrancyLock.
        // Note that callWithExactGas will revert if we do not have sufficient gas
        // to give the callee their requested amount.
        s_config.reentrancyLock = true;
        bool success = callWithExactGas(rc.callbackGasLimit, rc.sender, resp);
        s_config.reentrancyLock = false;
        return success;
    }

    function pay(
        RequestCommitment memory rc,
        bool isDirectPayment,
        uint256 startGas
    ) internal returns (uint256 payment) {
        // We want to charge users exactly for how much gas they use in their callback.
        // The gasAfterPaymentCalculation is meant to cover these additional operations where we
        // decrement the account balance and increment the oracles withdrawable balance.
        // We also add the flat KLAY fee to the payment amount.
        // Its specified in millionths of KLAY, if s_config.fulfillmentFlatFeeKlayPPM = 1
        // 1 KLAY / 1e6 = 1e18 pebs / 1e6 = 1e12 pebs.
        (uint256 balance, uint64 reqCount, , ) = s_prepayment.getAccount(rc.accId);

        if (isDirectPayment) {
            payment = balance;
        } else {
            payment = calculatePaymentAmount(
                startGas,
                s_config.gasAfterPaymentCalculation,
                getFeeTier(reqCount)
            );
        }

        s_prepayment.chargeFee(rc.accId, payment, msg.sender);

        return payment;
    }

    function fulfillDataRequestUint256(
        uint256 requestId,
        uint256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external nonReentrant returns (uint256) {
        uint256 startGas = gasleft();
        validateDataResponse(rc, requestId);
        RequestResponseConsumerBase rr;
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
        RequestResponseConsumerBase rr;
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
        RequestResponseConsumerBase rr;
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
        RequestResponseConsumerBase rr;
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
        RequestResponseConsumerBase rr;
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
        RequestResponseConsumerBase rr;
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
