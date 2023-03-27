import { Test, TestingModule } from '@nestjs/testing';
import { ReporterController } from './reporter.controller';
import { ReporterService } from './reporter.service';

describe('ReporterController', () => {
  let controller: ReporterController;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [ReporterController],
      providers: [ReporterService],
    }).compile();

    controller = module.get<ReporterController>(ReporterController);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });
});
