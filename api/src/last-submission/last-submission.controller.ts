import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { LastSubmissionService } from './last-submission.service'
import { LastSubmissionDto } from './dto/last-submission.dto'

@Controller({
  path: 'last-submission',
  version: '1'
})
export class LastSubmissionController {
  constructor(private readonly lastSubmissionService: LastSubmissionService) {}

  @Post()
  async create(@Body() lastSubmissionDto: LastSubmissionDto) {
    return this.lastSubmissionService.create(lastSubmissionDto)
  }

  @Get()
  async findAll() {
    return this.lastSubmissionService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return this.lastSubmissionService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() lastSubmissionDto: LastSubmissionDto) {
    return this.lastSubmissionService.update({
      where: { id: Number(id) },
      lastSubmissionDto
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return this.lastSubmissionService.remove({ id: Number(id) })
  }
}
