import { ApiProperty } from '@nestjs/swagger'

export class DatumDto {
  @ApiProperty()
  aggregator: number

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  value: number

  @ApiProperty()
  feed: number
}
