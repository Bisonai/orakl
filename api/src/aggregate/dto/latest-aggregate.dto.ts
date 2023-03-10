import { ApiProperty } from '@nestjs/swagger'

export class LatestAggregateDto {
  @ApiProperty()
  aggregatorHash: string
}
