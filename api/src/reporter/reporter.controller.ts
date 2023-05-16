import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { ReporterService } from './reporter.service'
import { CreateReporterDto } from './dto/create-reporter.dto'
import { UpdateReporterDto } from './dto/update-reporter.dto'

@Controller({
  path: 'reporter',
  version: '1'
})
export class ReporterController {
  constructor(private readonly reporterService: ReporterService) {}

  @Post()
  async create(@Body() createReporterDto: CreateReporterDto) {
    return await this.reporterService.create(createReporterDto)
  }

  @Get()
  async findAll(@Body('chain') chain: string, @Body('service') service: string) {
    return await this.reporterService.findAll({
      where: {
        chain: { name: chain },
        service: { name: service }
      }
    })
  }

  @Get('oracle-address/:oracleAddress')
  async findByOracleAddress(
    @Body('chain') chain: string,
    @Body('service') service: string,
    @Param('oracleAddress') oracleAddress: string
  ) {
    return await this.reporterService.findAll({
      where: {
        oracleAddress,
        chain: { name: chain },
        service: { name: service }
      }
    })
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.reporterService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() updateReporterDto: UpdateReporterDto) {
    return await this.reporterService.update({
      where: { id: Number(id) },
      updateReporterDto
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.reporterService.remove({ id: Number(id) })
  }
  @Post('encrypt/:text')
  async encrypt(@Param('text') text: string) {
    return await this.reporterService.encrypt(text)
  }
  @Post('decrypt/:text')
  async decrypt(@Param('text') text: string) {
    return await this.reporterService.decrypt(text)
  }
}
