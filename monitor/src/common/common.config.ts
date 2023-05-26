import { Injectable } from "@nestjs/common";
import { ConfigService } from "@nestjs/config";

@Injectable()
export class CommonConfigService {
  constructor(private readonly configService: ConfigService) {}

  get provider() {
    return this.configService.get<string>("common.provider");
  }
}
