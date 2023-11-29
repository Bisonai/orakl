import { Body, Controller, Delete, Get, Param, Patch, Post } from '@nestjs/common'
import { ReporterDto } from './dto/reporter.dto'
import { ReporterService } from './reporter.service'

@Controller({
  path: 'reporter',
  version: '1'
})
export class ReporterController {
  constructor(private readonly reporterService: ReporterService) {}

  @Post()
  create(@Body() reporterDto: ReporterDto) {
    return this.reporterService.create(reporterDto)
  }

  @Get()
  findAll() {
    return this.reporterService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.reporterService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  update(@Param('id') id: string, @Body() reporterDto: ReporterDto) {
    return this.reporterService.update({ where: { id: Number(id) }, reporterDto })
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.reporterService.remove({ id: Number(id) })
  }
}
