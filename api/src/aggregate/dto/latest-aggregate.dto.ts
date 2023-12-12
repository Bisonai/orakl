import { ApiProperty } from '@nestjs/swagger'

export class LatestAggregateDto {
  @ApiProperty()
  aggregatorHash: string
}

export class LatestAggregateByIdDto {
  @ApiProperty()
  aggregatorId: bigint
}
