/*
  Warnings:

  - A unique constraint covering the columns `[encodedName,contract_id]` on the table `functions` will be added. If there are existing duplicate values, this will fail.

*/
-- DropIndex
DROP INDEX "functions_encodedName_key";

-- CreateIndex
CREATE UNIQUE INDEX "functions_encodedName_contract_id_key" ON "functions"("encodedName", "contract_id");
