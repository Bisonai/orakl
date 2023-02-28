import * as Fs from "node:fs/promises";
import * as fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { ethers } from "ethers";
import * as dotenv from "dotenv";
dotenv.config();

export function add0x(s) {
  if (s.substring(0, 2) == "0x") {
    return s;
  } else {
    return "0x" + s;
  }
}

export function mkdir(dir: string) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

export async function readTextFile(filepath: string) {
  return await Fs.readFile(filepath, "utf8");
}

export async function writeTextFile(filepath: string, content: string) {
  await Fs.writeFile(filepath, content);
}
export async function writeTextAppend(filepath: string, content: string) {
  await Fs.appendFile(filepath, content);
}
export function mkTmpFile({ fileName }: { fileName: string }): string {
  const appPrefix = "orakl";
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), appPrefix));
  const tmpFilePath = path.join(tmpDir, fileName);
  return tmpFilePath;
}
export async function getTimestampByBlock(blockNumber: number) {
  let timestamp: number = 0;
  try {
    const provider = new ethers.providers.JsonRpcProvider(
      process.env.PROVIDER_URL
    );

    timestamp = (await provider.getBlock(blockNumber)).timestamp;
  } catch (error) {
    await getTimestampByBlock(blockNumber);
  }
  return timestamp;
}
