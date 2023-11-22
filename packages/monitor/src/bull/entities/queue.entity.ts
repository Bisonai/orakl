import { OmitType } from '@nestjs/mapped-types';

export class QueueDto {
  idx: number;
  service: string;
  name: string;
  status: boolean
}

export class CreateQueueDto extends OmitType(QueueDto, ['idx'] as const) { }
export class QueueUpdateDto extends OmitType(QueueDto, ['idx'] as const) {}
