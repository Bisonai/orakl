import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger'

export class ProxyDto {
  @ApiProperty()
  protocol: string

  @ApiProperty()
  host: string

  @ApiProperty()
  port: number

  @ApiProperty()
  location?: string
}
