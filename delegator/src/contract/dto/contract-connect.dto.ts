import { ApiProperty } from '@nestjs/swagger'

export class ContractConnectDto {
  @ApiProperty()
  contractId: bigint

  @ApiProperty()
  reporterId: bigint
}
