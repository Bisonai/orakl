import * as Fs from 'node:fs/promises';
import * as fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
export async function loadJson(filepath) {
    const json = await Fs.readFile(filepath, 'utf8');
    return JSON.parse(json);
}
export const pipe = (...fns) => (x) => fns.reduce((v, f) => f(v), x);
export function remove0x(s) {
    if (s.substring(0, 2) == '0x') {
        return s.substring(2);
    }
}
export function add0x(s) {
    if (s.substring(0, 2) == '0x') {
        return s;
    }
    else {
        return '0x' + s;
    }
}
export function pad32Bytes(data) {
    data = remove0x(data);
    let s = String(data);
    while (s.length < (64 || 2)) {
        s = '0' + s;
    }
    return s;
}
export function mkdir(dir) {
    if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
    }
}
export async function readTextFile(filepath) {
    return await Fs.readFile(filepath, 'utf8');
}
export async function writeTextFile(filepath, content) {
    await Fs.writeFile(filepath, content);
}
export function mkTmpFile({ fileName }) {
    const appPrefix = 'orakl';
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), appPrefix));
    const tmpFilePath = path.join(tmpDir, fileName);
    return tmpFilePath;
}
//# sourceMappingURL=utils.js.map