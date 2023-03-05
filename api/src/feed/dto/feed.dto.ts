import { ApiProperty } from '@nestjs/swagger'

export class FeedDto {
  @ApiProperty()
  name: string

  @ApiProperty()
  definition: string
}
