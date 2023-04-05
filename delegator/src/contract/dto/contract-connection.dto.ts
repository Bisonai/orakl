import { ApiProperty } from '@nestjs/swagger'

export class ContractConnectionDto {
  @ApiProperty()
  contractId: bigint

  @ApiProperty()
  reporterId: bigint
}
