import { Injectable, UnauthorizedException } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { PASSWORD } from 'src/common/configuration';

@Injectable()
export class AuthService {
  constructor(
    private jwtService: JwtService
  ) {}

  async signIn(pass) {
    if (PASSWORD !== pass) {
      throw new UnauthorizedException();
    }
    const payload = { password: pass};
    return {
      access_token: await this.jwtService.signAsync(payload),
    };
  }
}
