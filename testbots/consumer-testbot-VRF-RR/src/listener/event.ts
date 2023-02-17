import * as path from "node:path";
import { Contract, ethers } from "ethers";
import { IListenerBlock, IListenerConfig } from "../types";
import { mkdir, readTextFile, writeTextFile } from "../utils";
import * as dotenv from "dotenv";
dotenv.config();

const PROVIDER_URL = process.env.PROVIDER_URL;
const LISTENER_ROOT_DIR = "./tmp/listener/";
const LISTENER_DELAY = 1000;
export class Event {
  fn: (log) => void;
  emitContract: Contract;
  listenerBlock: IListenerBlock;
  provider: ethers.providers.JsonRpcProvider;
  eventName: string;
  running: boolean;

  constructor(
    wrapFn: (iface: ethers.utils.Interface) => (log) => void,
    abi,
    listener: IListenerConfig
  ) {
    mkdir(LISTENER_ROOT_DIR);
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL);
    const iface = new ethers.utils.Interface(abi);

    this.running = false;
    this.provider = provider;
    this.fn = wrapFn(iface);
    this.eventName = listener.eventName;
    this.emitContract = new ethers.Contract(listener.address, abi, provider);
    this.listenerBlock = {
      startBlock: 0,
      filePath: path.join(LISTENER_ROOT_DIR, `${listener.address}.txt`),
    };
  }

  listen() {
    setInterval(async () => {
      if (!this.running) {
        this.running = true;
        await this.filter();
        this.running = false;
      } else {
        console.log({ name: "Event:listen" }, "running");
      }
    }, LISTENER_DELAY);
  }

  async filter() {
    try {
      if (this.listenerBlock.startBlock == 0) {
        try {
          this.listenerBlock.startBlock = parseInt(
            await readTextFile(this.listenerBlock.filePath)
          );
        } catch {
          this.listenerBlock.startBlock = await this.getLatestBlock();
        }
      }

      const latestBlock = await this.getLatestBlock();
      if (latestBlock >= this.listenerBlock.startBlock) {
        const events = await this.emitContract.queryFilter(
          this.eventName,
          this.listenerBlock.startBlock,
          latestBlock
        );

        console.debug(
          { name: "Event:filter" },
          `${this.listenerBlock.startBlock}-${latestBlock}`
        );
        this.listenerBlock.startBlock = latestBlock + 1;
        await writeTextFile(
          this.listenerBlock.filePath,
          this.listenerBlock.startBlock.toString()
        );

        if (events?.length > 0) {
          for await (const event of events) {
            this.fn(event);
          }
        }
      }
    } catch (e) {
      console.log({ name: "Event:filter" }, e);
    }
  }

  async getLatestBlock() {
    return await this.provider.getBlockNumber();
  }
}
