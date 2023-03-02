import { ApiProperty } from '@nestjs/swagger'

export class CreateChainDto {
  @ApiProperty()
  name: string
}
