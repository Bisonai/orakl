import { Test, TestingModule } from '@nestjs/testing';
import { SignService } from './sign.service';

describe('SignService', () => {
  let service: SignService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [SignService],
    }).compile();

    service = module.get<SignService>(SignService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });
});
