import { ApiProperty } from '@nestjs/swagger'

export class AggregatorWhereDto {
  @ApiProperty()
  active?: boolean

  @ApiProperty()
  chain?: string
}
