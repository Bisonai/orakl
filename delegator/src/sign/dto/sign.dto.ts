import { ApiProperty } from '@nestjs/swagger'

export class SignDto {
  @ApiProperty()
  tx: string
  signed: string
}
