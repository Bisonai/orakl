import { ApiProperty } from '@nestjs/swagger'

export class ErrorDto {
  @ApiProperty()
  requestId: string

  @ApiProperty()
  timestamp: string | Date

  @ApiProperty()
  code: number

  @ApiProperty()
  name: string

  @ApiProperty()
  stack: string
}
