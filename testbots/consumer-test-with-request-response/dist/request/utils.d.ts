import { ethers } from "ethers";
export declare function buildWallet(): ethers.Wallet | undefined;
export declare function sendTransaction(wallet: any, to: any, payload: any, gasLimit?: any, value?: any): Promise<any>;
