import { Test, TestingModule } from '@nestjs/testing';
import { AdapterController } from './adapter.controller';
import { AdapterService } from './adapter.service';

describe('AdapterController', () => {
  let controller: AdapterController;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [AdapterController],
      providers: [AdapterService],
    }).compile();

    controller = module.get<AdapterController>(AdapterController);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });
});
