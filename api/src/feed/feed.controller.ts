import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { FeedService } from './feed.service'
import { CreateFeedDto } from './dto/create-feed.dto'
import { UpdateFeedDto } from './dto/update-feed.dto'

@Controller('feed')
export class FeedController {
  constructor(private readonly feedService: FeedService) {}

  @Get()
  findAll() {
    return this.feedService.findAll()
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.feedService.findOne(+id)
  }
}
