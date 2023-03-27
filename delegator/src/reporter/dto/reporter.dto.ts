import { ApiProperty } from '@nestjs/swagger'

export class ReporterDto {
  @ApiProperty()
  address: string

  @ApiProperty()
  contractId: number

  @ApiProperty()
  organizationId: number
}
