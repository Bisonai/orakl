import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { ContractService } from './contract.service'
import { ContractConnectionDto } from './dto/contract-connection.dto'
import { ContractDto } from './dto/contract.dto'

@Controller({
  path: 'contract',
  version: '1'
})
export class ContractController {
  constructor(private readonly contractService: ContractService) {}

  @Post()
  create(@Body() contractDto: ContractDto) {
    return this.contractService.create(contractDto)
  }

  @Get()
  findAll() {
    return this.contractService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.contractService.findOne({ id: Number(id) })
  }

  @Post('/connectReporter')
  connectReporter(@Body() contractConnectionDto: ContractConnectionDto) {
    return this.contractService.connectReporter(
      contractConnectionDto.contractId,
      contractConnectionDto.reporterId
    )
  }

  @Post('/disconnectReporter')
  disconnectReporter(@Body() contractConnectionDto: ContractConnectionDto) {
    return this.contractService.disconnectReporter(
      contractConnectionDto.contractId,
      contractConnectionDto.reporterId
    )
  }

  @Patch(':id')
  update(@Param('id') id: string, @Body() contractDto: ContractDto) {
    return this.contractService.update({ where: { id: Number(id) }, contractDto })
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.contractService.remove({ id: Number(id) })
  }
}
