import { Injectable } from "@nestjs/common";
import { SERVICE } from "src/common/types";
import { MonitorConfigRepository } from "./monitor.config.repository";

@Injectable()
export class MonitorConfigService {
  constructor(private readonly monitorConfigRepository: MonitorConfigRepository) {}

  async registerConfig(name, value) {
    return await this.monitorConfigRepository.createConfig(name, value)
  }

  async getValueByName(name) {
    return await this.monitorConfigRepository.getConfigByName(name)
  }    
    
}
