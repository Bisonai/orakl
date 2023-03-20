import { ApiProperty } from '@nestjs/swagger'

export class AggregatorUpdateDto {
  @ApiProperty()
  chain: string
}
