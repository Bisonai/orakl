import { ApiProperty } from '@nestjs/swagger'

export class ReporterDto {
  @ApiProperty()
  address: string

  @ApiProperty()
  organizationId: bigint
}
