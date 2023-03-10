import { Controller, Get, Param } from '@nestjs/common'
import { FeedService } from './feed.service'

@Controller({
  path: 'feed',
  version: '1'
})
export class FeedController {
  constructor(private readonly feedService: FeedService) {}

  @Get()
  async findAll() {
    return await this.feedService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.feedService.findOne({ id: Number(id) })
  }
}
