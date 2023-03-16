import { ApiProperty } from '@nestjs/swagger'

export class UpdateListenerDto {
  @ApiProperty()
  address: string

  @ApiProperty()
  eventName: string
}
