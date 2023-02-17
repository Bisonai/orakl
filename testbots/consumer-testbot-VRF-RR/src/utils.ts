import * as Fs from "node:fs/promises";
import * as fs from "node:fs";
import os from "node:os";
import path from "node:path";

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
