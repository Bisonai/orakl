import { ApiProperty } from '@nestjs/swagger'

export class FeedDto {
  @ApiProperty()
  source: string

  @ApiProperty()
  decimals: number

  @ApiProperty()
  latestRound: number

  @ApiProperty()
  definition: string
}
