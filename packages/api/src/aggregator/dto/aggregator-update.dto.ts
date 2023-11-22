import { ApiProperty } from '@nestjs/swagger'

export class AggregatorUpdateDto {
  @ApiProperty()
  active: boolean

  @ApiProperty()
  chain: string
}
