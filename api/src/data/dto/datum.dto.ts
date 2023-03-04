import { ApiProperty } from '@nestjs/swagger'

export class DatumDto {
  @ApiProperty()
  round: number

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  value: number

  @ApiProperty()
  aggregator: number

  @ApiProperty()
  feed: number
}
