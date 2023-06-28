import { Transform, Type } from "class-transformer";
import { IsArray, IsString } from "class-validator";

export class RquestDto {
  @IsString()
  @Transform(({ value }) =>
    value
      .slice(1, -1)
      .split(",")
      .map((element) => `'${element.trim()}'`)
      .join(",")
  ) // Custom transformation decorator
  query: string;
}
