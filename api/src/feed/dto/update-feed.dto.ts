import { PartialType } from '@nestjs/swagger'
import { CreateFeedDto } from './create-feed.dto'

export class UpdateFeedDto extends PartialType(CreateFeedDto) {}
