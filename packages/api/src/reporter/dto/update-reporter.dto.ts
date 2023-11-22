import { ApiProperty } from '@nestjs/swagger'

export class UpdateReporterDto {
  @ApiProperty()
  address: string

  @ApiProperty()
  privateKey: string

  @ApiProperty()
  oracleAddress: string
}
