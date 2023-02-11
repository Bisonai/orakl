import { Contract, ethers } from "ethers";
import { Logger } from "pino";
import { IListenerBlock, IListenerConfig } from "../types";
export declare class Event {
    fn: (log: any) => void;
    emitContract: Contract;
    listenerBlock: IListenerBlock;
    provider: ethers.providers.JsonRpcProvider;
    eventName: string;
    running: boolean;
    logger: Logger;
    constructor(wrapFn: (iface: ethers.utils.Interface) => (log: any) => void, abi: any, listener: IListenerConfig);
    listen(): void;
    filter(): Promise<void>;
    getLatestBlock(): Promise<number>;
}
