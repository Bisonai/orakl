import { ApiProperty } from '@nestjs/swagger'

export class CreateListenerDto {
  @ApiProperty()
  address: string

  @ApiProperty()
  eventName: string

  @ApiProperty()
  chain: string

  @ApiProperty()
  service: string
}
