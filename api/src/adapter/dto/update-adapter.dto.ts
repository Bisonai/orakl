import { PartialType } from '@nestjs/swagger';
import { CreateAdapterDto } from './create-adapter.dto';

export class UpdateAdapterDto extends PartialType(CreateAdapterDto) {}
