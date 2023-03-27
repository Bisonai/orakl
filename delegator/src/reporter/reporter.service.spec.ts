import { Test, TestingModule } from '@nestjs/testing';
import { ReporterService } from './reporter.service';

describe('ReporterService', () => {
  let service: ReporterService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ReporterService],
    }).compile();

    service = module.get<ReporterService>(ReporterService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
