/*
  Warnings:

  - You are about to drop the column `ach_transfer_id` on the `payment` table. All the data in the column will be lost.
  - You are about to drop the column `rpt_transfer_id` on the `payment` table. All the data in the column will be lost.
  - Added the required column `payment_rail` to the `payment` table without a default value. This is not possible if the table is not empty.
  - Added the required column `name` to the `recipient` table without a default value. This is not possible if the table is not empty.
  - Added the required column `account_number` to the `transfer` table without a default value. This is not possible if the table is not empty.
  - Added the required column `routing_number` to the `transfer` table without a default value. This is not possible if the table is not empty.

*/
-- CreateEnum
CREATE TYPE "payment_rail" AS ENUM ('ach', 'rtp');

-- AlterTable
ALTER TABLE "payment" DROP COLUMN "ach_transfer_id",
DROP COLUMN "rpt_transfer_id",
ADD COLUMN     "bank_transfer_id" VARCHAR(255),
ADD COLUMN     "payment_rail" "payment_rail" NOT NULL,
ALTER COLUMN "chariot_id" DROP NOT NULL;

-- AlterTable
ALTER TABLE "recipient" ADD COLUMN     "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD COLUMN     "name" VARCHAR(255) NOT NULL;

-- AlterTable
ALTER TABLE "transfer" ADD COLUMN     "account_number" VARCHAR(255) NOT NULL,
ADD COLUMN     "routing_number" VARCHAR(255) NOT NULL;
