import { ApiProperty } from '@nestjs/swagger'

export class AggregatorDto {
  @ApiProperty()
  aggregatorHash: string
}
