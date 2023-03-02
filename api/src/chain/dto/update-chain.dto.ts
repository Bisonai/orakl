import { ApiProperty } from '@nestjs/swagger'

export class UpdateChainDto {
  @ApiProperty()
  name: string
}
