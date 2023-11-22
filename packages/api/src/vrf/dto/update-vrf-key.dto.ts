import { ApiProperty } from '@nestjs/swagger'

export class UpdateVrfKeyDto {
  @ApiProperty()
  sk: string

  @ApiProperty()
  pk: string

  @ApiProperty()
  pkX: string

  @ApiProperty()
  pkY: string

  @ApiProperty()
  keyHash: string
}
