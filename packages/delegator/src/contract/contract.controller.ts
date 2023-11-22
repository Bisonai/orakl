import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { ContractService } from './contract.service'
import { ContractConnectDto } from './dto/contract-connect.dto'
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
  connectReporter(@Body() contractConnectionDto: ContractConnectDto) {
    return this.contractService.connectReporter(contractConnectionDto)
  }

  @Post('/disconnectReporter')
  disconnectReporter(@Body() contractConnectionDto: ContractConnectDto) {
    return this.contractService.disconnectReporter(contractConnectionDto)
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
