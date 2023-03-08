import { ApiProperty } from '@nestjs/swagger'

export class DatumDto {
  @ApiProperty()
  aggregatorId: bigint

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  value: number

  @ApiProperty()
  feedId: bigint
}
