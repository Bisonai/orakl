import { Test, TestingModule } from '@nestjs/testing';
import { AdapterService } from './adapter.service';

describe('AdapterService', () => {
  let service: AdapterService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [AdapterService],
    }).compile();

    service = module.get<AdapterService>(AdapterService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
