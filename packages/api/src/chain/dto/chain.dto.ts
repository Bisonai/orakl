import { ApiProperty } from '@nestjs/swagger'

export class ChainDto {
  @ApiProperty()
  name: string
}
