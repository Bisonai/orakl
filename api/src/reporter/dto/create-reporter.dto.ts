import { ApiProperty } from '@nestjs/swagger'

export class CreateReporterDto {
  @ApiProperty()
  address: string

  @ApiProperty()
  privateKey: string

  @ApiProperty()
  oracleAddress: string

  @ApiProperty()
  chain: string //| bigint

  @ApiProperty()
  service: string //| bigint
}
