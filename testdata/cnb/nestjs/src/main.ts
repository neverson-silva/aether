import "reflect-metadata";
import { NestFactory } from "@nestjs/core";
import { Controller, Get, Module } from "@nestjs/common";

@Controller()
class AppController {
  @Get()
  root() {
    return "aether nestjs fixture";
  }
}

@Module({ controllers: [AppController] })
class AppModule {}

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  await app.listen(process.env.PORT || 8080);
}
void bootstrap();
