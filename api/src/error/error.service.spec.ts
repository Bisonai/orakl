import { Test, TestingModule } from '@nestjs/testing';
import { ErrorService } from './error.service';

describe('ErrorService', () => {
  let service: ErrorService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ErrorService],
    }).compile();

    service = module.get<ErrorService>(ErrorService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
