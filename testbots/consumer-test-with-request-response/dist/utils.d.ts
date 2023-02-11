export declare function loadJson(filepath: any): Promise<any>;
export declare const pipe: (...fns: any[]) => (x: any) => any;
export declare function remove0x(s: any): any;
export declare function add0x(s: any): any;
export declare function pad32Bytes(data: any): string;
export declare function mkdir(dir: string): void;
export declare function readTextFile(filepath: string): Promise<string>;
export declare function writeTextFile(filepath: string, content: string): Promise<void>;
export declare function mkTmpFile({ fileName }: {
    fileName: string;
}): string;
