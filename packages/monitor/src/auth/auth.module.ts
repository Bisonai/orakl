import { Module } from '@nestjs/common';
import { JwtModule } from '@nestjs/jwt';
import { AuthService } from './auth.service';
import { jwtConstants } from 'src/common/configuration';
import { AuthController } from './auth.controller';


@Module({
  imports: [
    JwtModule.register({
      secret: jwtConstants.secret, // Replace with your own secret key
      signOptions: { expiresIn: '1h' }, // Set the token expiration time as desired
    }),
  ],
  controllers: [AuthController],
  providers: [
    AuthService
    ],
    exports: [AuthService],
})
export class AuthModule {}
