import { ApiProperty } from '@nestjs/swagger'

export class AggregateDto {
  @ApiProperty()
  aggregatorId: bigint

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  value: number
}
