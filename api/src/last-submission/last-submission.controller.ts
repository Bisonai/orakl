import { Controller, Get, Body, Param, Put } from '@nestjs/common'
import { LastSubmissionService } from './last-submission.service'
import { LastSubmissionDto } from './dto/last-submission.dto'

@Controller({
  path: 'last-submission',
  version: '1'
})
export class LastSubmissionController {
  constructor(private readonly lastSubmissionService: LastSubmissionService) {}

  @Get(':hash')
  async findByhash(@Param('hash') aggregatorHash: string) {
    return await this.lastSubmissionService.findByhash({ aggregatorHash })
  }

  @Put()
  async upsert(@Body() data: LastSubmissionDto) {
    return await this.lastSubmissionService.upsert(data)
  }
}
