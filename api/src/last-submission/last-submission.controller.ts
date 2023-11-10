import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { LastSubmissionService } from './last-submission.service'
import { LastSubmissionDto } from './dto/last-submission.dto'

@Controller({
  path: 'last-submission',
  version: '1'
})
export class LastSubmissionController {
  constructor(private readonly lastSubmissionService: LastSubmissionService) {}

  @Get(':hash/latest')
  async findByhash(@Param('hash') aggregatorHash: string) {
    return await this.lastSubmissionService.findByhash({ aggregatorHash })
  }

  @Post('upsert')
  async upsert(@Body() lastSubmissionDto: LastSubmissionDto) {
    return this.lastSubmissionService.upsert(lastSubmissionDto)
  }
}
