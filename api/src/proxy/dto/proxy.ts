import { ApiProperty } from '@nestjs/swagger'

export class ProxyDto {
  @ApiProperty()
  protocol: string

  @ApiProperty()
  host: string

  @ApiProperty()
  port: string
}
