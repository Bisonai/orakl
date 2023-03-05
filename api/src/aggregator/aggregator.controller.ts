import {
  Controller,
  Get,
  Post,
  Body,
  Delete,
  Param,
  HttpException,
  HttpStatus
} from '@nestjs/common'
import { Aggregator as AggregatorModel } from '@prisma/client'
import { AggregatorService } from './aggregator.service'
import { AggregatorDto } from './dto/aggregator.dto'
import { AggregatorWhereDto } from './dto/aggregator-where.dto'
import { PRISMA_ERRORS } from '../errors'

@Controller({
  path: 'aggregator',
  version: '1'
})
export class AggregatorController {
  constructor(private readonly aggregatorService: AggregatorService) {}

  @Post()
  async create(@Body() aggregatorDto: AggregatorDto) {
    return await this.aggregatorService.create(aggregatorDto).catch((err) => {
      throw new HttpException(
        {
          message: PRISMA_ERRORS[err.code](err.meta)
        },
        HttpStatus.BAD_REQUEST
      )
    })
  }

  @Get()
  findAll(@Body() whereDto: AggregatorWhereDto) {
    return this.aggregatorService.findAll({
      where: {
        chain: { name: whereDto.chain },
        active: whereDto.active
      }
    })
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.aggregatorService.findOne({ id: Number(id) })
  }

  @Delete(':id')
  async remove(@Param('id') id: string): Promise<AggregatorModel> {
    return this.aggregatorService.remove({ id: Number(id) })
  }
}
