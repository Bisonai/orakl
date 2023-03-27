import { Test, TestingModule } from '@nestjs/testing';
import { MethodService } from './method.service';

describe('MethodService', () => {
  let service: MethodService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [MethodService],
    }).compile();

    service = module.get<MethodService>(MethodService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
