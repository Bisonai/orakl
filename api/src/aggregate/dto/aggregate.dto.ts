import { ApiProperty } from '@nestjs/swagger'

export class AggregateDto {
  @ApiProperty()
  aggregatorId: number

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  value: number
}
