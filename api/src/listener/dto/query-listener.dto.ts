import { ApiProperty } from '@nestjs/swagger'

export class QueryListenerDto {
  @ApiProperty()
  chain?: string

  @ApiProperty()
  service?: string
}
