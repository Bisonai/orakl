import { PartialType } from '@nestjs/swagger';
import { CreateAggregatorDto } from './create-aggregator.dto';

export class UpdateAggregatorDto extends PartialType(CreateAggregatorDto) {}
