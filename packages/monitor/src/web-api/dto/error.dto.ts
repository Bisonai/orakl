// account.dto.ts
export class ErrorResultDto {
    error_id: bigint
    request_id: string;
    timestamp: bigint;
    code: number;
    name: string;
    stack: string;
}
