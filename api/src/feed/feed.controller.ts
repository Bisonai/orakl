import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { FeedService } from './feed.service'
import { CreateFeedDto } from './dto/create-feed.dto'
import { UpdateFeedDto } from './dto/update-feed.dto'

@Controller('feed')
export class FeedController {
  constructor(private readonly feedService: FeedService) {}

  @Post()
  create(@Body() createFeedDto: CreateFeedDto) {
    return this.feedService.create(createFeedDto)
  }

  @Get()
  findAll() {
    return this.feedService.findAll()
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.feedService.findOne(+id)
  }

  @Patch(':id')
  update(@Param('id') id: string, @Body() updateFeedDto: UpdateFeedDto) {
    return this.feedService.update(+id, updateFeedDto)
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.feedService.remove(+id)
  }
}
