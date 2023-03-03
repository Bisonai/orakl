import { Controller, Get, Param } from '@nestjs/common'
import { FeedService } from './feed.service'

@Controller({
  path: 'feed',
  version: '1'
})
export class FeedController {
  constructor(private readonly feedService: FeedService) {}

  @Get()
  findAll() {
    return this.feedService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.feedService.findOne({ id: Number(id) })
  }
}
