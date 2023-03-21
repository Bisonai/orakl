interface IDirectPaymentConfig {
  fulfillmentFee: string
  baseFee: string
}

interface IFeeConfig {
  fulfillmentFlatFeeKlayPPMTier1: number
  fulfillmentFlatFeeKlayPPMTier2: number
  fulfillmentFlatFeeKlayPPMTier3: number
  fulfillmentFlatFeeKlayPPMTier4: number
  fulfillmentFlatFeeKlayPPMTier5: number
  reqsForTier2: number
  reqsForTier3: number
  reqsForTier4: number
  reqsForTier5: number
}

// Aggregator
interface IAggregatorDeployConfig {
  name: string
  paymentAmount: number
  timeout: number
  validator: string
  decimals: number
  description: string
  depositAmount?: number
}

interface IAggregatorChangeOraclesConfig {
  removed: string[]
  added: string[]
  addedAdmins: string[]
  minSubmissionCount: number
  maxSubmissionCount: number
  restartDelay: number
  aggregatorAddress?: string
}

export interface IAggregatorConfig {
  deploy?: IAggregatorDeployConfig
  changeOracles?: IAggregatorChangeOraclesConfig
}

export interface ICoordinatorConfig {
  maxGasLimit: number
  gasAfterPaymentCalculation: number
  feeConfig: IFeeConfig
}

// Coordinator
interface ICoordinatorDeploy {
  version: string
}

export interface ICoordinatorMinBalance {
  minBalance: string
}

export interface IAddCoordinator {
  prepaymentAddress: string
  coordinatorAddress: string
}

// RequestResponseCoordinator
export interface ICoordinatorDirectPaymentConfig {
  directPaymentConfig: IDirectPaymentConfig
}

export interface IRRCConfig {
  requestResponseCoordinatorAddress?: string
  deploy?: ICoordinatorDeploy
  registerOracle?: string[]
  deregisterOracle?: string[]
  setConfig?: ICoordinatorConfig
  setDirectPaymentConfig?: ICoordinatorDirectPaymentConfig
  setMinBalance?: ICoordinatorMinBalance
  addCoordinator?: IAddCoordinator
}

// VRFCoordinator
interface IRegisterProvingKey {
  address: string
  publicProvingKey: [string, string]
}

interface IDeregisterProvingKey {
  publicProvingKey: [string, string]
}

export interface IVRFCoordinatorConfig {
  vrfCoordinatorAddress?: string
  deploy?: ICoordinatorDeploy
  registerProvingKey?: IRegisterProvingKey[]
  deregisterProvingKey?: IDeregisterProvingKey[]
  setConfig?: ICoordinatorConfig
  setDirectPaymentConfig?: ICoordinatorDirectPaymentConfig
  setMinBalance?: ICoordinatorMinBalance
  addCoordinator?: IAddCoordinator
}
