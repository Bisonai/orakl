-- CreateTable
CREATE TABLE "VrfKey" (
    "id" BIGSERIAL NOT NULL,
    "sk" VARCHAR(64) NOT NULL,
    "pk" VARCHAR(130) NOT NULL,
    "pkX" VARCHAR(77) NOT NULL,
    "pkY" VARCHAR(77) NOT NULL,
    "keyHash" VARCHAR(66) NOT NULL,
    "chainId" BIGINT NOT NULL,

    CONSTRAINT "VrfKey_pkey" PRIMARY KEY ("id")
);

-- AddForeignKey
ALTER TABLE "VrfKey" ADD CONSTRAINT "VrfKey_chainId_fkey" FOREIGN KEY ("chainId") REFERENCES "Chain"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
