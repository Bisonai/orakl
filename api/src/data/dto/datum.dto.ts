import { ApiProperty } from '@nestjs/swagger'

export class DatumDto {
  @ApiProperty()
  aggregatorId: number

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  value: number

  @ApiProperty()
  feed: number
}
