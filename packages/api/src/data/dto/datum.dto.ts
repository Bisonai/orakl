import { ApiProperty } from '@nestjs/swagger'

export class DatumDto {
  @ApiProperty()
  aggregatorId: bigint

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  value: bigint

  @ApiProperty()
  feedId: bigint
}
