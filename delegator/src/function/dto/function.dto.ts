import { ApiProperty } from '@nestjs/swagger'

export class FunctionDto {
  @ApiProperty()
  name: string

  @ApiProperty()
  contractId: number

  @ApiProperty()
  encodedName: string
}
