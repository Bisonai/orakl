import { Test, TestingModule } from '@nestjs/testing';
import { ChainController } from './chain.controller';
import { ChainService } from './chain.service';

describe('ChainController', () => {
  let controller: ChainController;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [ChainController],
      providers: [ChainService],
    }).compile();

    controller = module.get<ChainController>(ChainController);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });
});
