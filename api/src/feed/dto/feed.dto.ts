import { ApiProperty } from '@nestjs/swagger'

export class FeedDto {
  @ApiProperty()
  name: string

  @ApiProperty()
  latestRound: number

  @ApiProperty()
  definition: string
}
