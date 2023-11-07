-- CreateEnum
CREATE TYPE "address_status" AS ENUM ('active', 'inactive');

-- CreateEnum
CREATE TYPE "bank_address_status" AS ENUM ('active', 'inactive');

-- CreateTable
CREATE TABLE "user" (
    "id" BIGSERIAL NOT NULL,
    "email" VARCHAR(255) NOT NULL,

    CONSTRAINT "user_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "idempotency_key" (
    "id" BIGSERIAL NOT NULL,
    "key" VARCHAR(255) NOT NULL,
    "last_run_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "locked_at" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "request_method" VARCHAR(10) NOT NULL,
    "request_path" VARCHAR(255) NOT NULL,
    "request_params" JSONB,
    "response_code" INTEGER,
    "response_body" JSONB,
    "recovery_point" VARCHAR(255) NOT NULL,
    "user_id" BIGINT NOT NULL,

    CONSTRAINT "idempotency_key_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "audit_record" (
    "id" BIGSERIAL NOT NULL,
    "action" VARCHAR(50) NOT NULL,
    "data" JSONB NOT NULL,
    "origin_ip" VARCHAR(50) NOT NULL,
    "resource_type" VARCHAR(50) NOT NULL,
    "resource_id" BIGINT NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "user_id" BIGINT NOT NULL,

    CONSTRAINT "audit_record_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "account" (
    "id" BIGSERIAL NOT NULL,
    "bank_account_id" VARCHAR(255),
    "bank_account_number_id" VARCHAR(255),
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "idempotency_key_id" BIGINT,
    "user_id" BIGINT NOT NULL,

    CONSTRAINT "account_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "organization" (
    "id" VARCHAR(255) NOT NULL,
    "legal_name" VARCHAR(255) NOT NULL,
    "preferred_name" VARCHAR(255),
    "ein" VARCHAR(9) NOT NULL,
    "address_id" BIGINT NOT NULL,

    CONSTRAINT "organization_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "recipient" (
    "id" UUID NOT NULL,
    "primary" BOOLEAN NOT NULL DEFAULT false,
    "organization_id" VARCHAR(255) NOT NULL,
    "bank_address_id" BIGINT NOT NULL,
    "mailing_address_id" BIGINT NOT NULL,

    CONSTRAINT "recipient_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "payment" (
    "id" BIGSERIAL NOT NULL,
    "amount" BIGINT NOT NULL,
    "description" VARCHAR(400) NOT NULL,
    "ach_transfer_id" VARCHAR(255),
    "rpt_transfer_id" VARCHAR(255),
    "account_id" BIGINT NOT NULL,
    "recipient_id" UUID NOT NULL,
    "chariot_id" VARCHAR(255) NOT NULL,
    "idempotency_key_id" BIGINT,
    "user_id" BIGINT NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "payment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "transfer" (
    "id" BIGSERIAL NOT NULL,
    "amount" BIGINT NOT NULL,
    "description" VARCHAR(400) NOT NULL,
    "ach_transfer_id" VARCHAR(255),
    "account_id" BIGINT NOT NULL,
    "idempotency_key_id" BIGINT,
    "user_id" BIGINT NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "transfer_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "address" (
    "id" BIGSERIAL NOT NULL,
    "line1" VARCHAR(255) NOT NULL,
    "line2" VARCHAR(255),
    "city" VARCHAR(255) NOT NULL,
    "state" VARCHAR(10) NOT NULL,
    "postalCode" VARCHAR(20) NOT NULL,
    "status" "address_status" NOT NULL,
    "updated_at" TIMESTAMPTZ(6) NOT NULL,

    CONSTRAINT "address_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "bank_address" (
    "id" BIGSERIAL NOT NULL,
    "account_number" VARCHAR(255) NOT NULL,
    "routing_number" VARCHAR(255) NOT NULL,
    "status" "bank_address_status" NOT NULL,
    "updated_at" TIMESTAMPTZ(6) NOT NULL,

    CONSTRAINT "bank_address_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "user_email_key" ON "user"("email");

-- CreateIndex
CREATE UNIQUE INDEX "idempotency_key_key_key" ON "idempotency_key"("key");

-- CreateIndex
CREATE INDEX "idx_idempotency_keys_user_id_key" ON "idempotency_key"("user_id", "key");

-- CreateIndex
CREATE UNIQUE INDEX "account_bank_account_id_key" ON "account"("bank_account_id");

-- CreateIndex
CREATE UNIQUE INDEX "account_bank_account_number_id_key" ON "account"("bank_account_number_id");

-- CreateIndex
CREATE INDEX "idx_account_idempotency_key_id" ON "account"("idempotency_key_id");

-- CreateIndex
CREATE UNIQUE INDEX "idx_account_user_id_idempotency_key_id" ON "account"("user_id", "idempotency_key_id");

-- CreateIndex
CREATE UNIQUE INDEX "organization_ein_key" ON "organization"("ein");

-- CreateIndex
CREATE INDEX "idx_recipient_organization_id" ON "recipient"("organization_id");

-- CreateIndex
CREATE INDEX "idx_payment_idempotency_key_id" ON "payment"("idempotency_key_id");

-- CreateIndex
CREATE INDEX "idx_payment_account_id" ON "payment"("account_id");

-- CreateIndex
CREATE INDEX "idx_payment_chariot_id" ON "payment"("chariot_id");

-- CreateIndex
CREATE INDEX "idx_payment_account_id_recipient_id" ON "payment"("account_id", "recipient_id");

-- CreateIndex
CREATE UNIQUE INDEX "idx_payment_user_id_idempotency_key_id" ON "payment"("user_id", "idempotency_key_id");

-- CreateIndex
CREATE INDEX "idx_transfer_idempotency_key_id" ON "transfer"("idempotency_key_id");

-- CreateIndex
CREATE INDEX "idx_transfer_account_id" ON "transfer"("account_id");

-- CreateIndex
CREATE UNIQUE INDEX "idx_transfer_user_id_idempotency_key_id" ON "transfer"("user_id", "idempotency_key_id");

-- CreateIndex
CREATE INDEX "idx_address_line1_line2_city_state" ON "address"("line1", "line2", "city", "state");

-- CreateIndex
CREATE INDEX "idx_address_line1_line2_postal_code" ON "address"("line1", "line2", "postalCode");

-- CreateIndex
CREATE INDEX "idx_bank_address_routing_number_account_number" ON "bank_address"("routing_number", "account_number");

-- AddForeignKey
ALTER TABLE "idempotency_key" ADD CONSTRAINT "fk_idempotency_key_user" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "audit_record" ADD CONSTRAINT "fk_audit_record_user" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "account" ADD CONSTRAINT "fk_account_idempotency_key" FOREIGN KEY ("idempotency_key_id") REFERENCES "idempotency_key"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "account" ADD CONSTRAINT "fk_account_user" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization" ADD CONSTRAINT "fk_organization_address" FOREIGN KEY ("address_id") REFERENCES "address"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "recipient" ADD CONSTRAINT "fk_recipient_organization" FOREIGN KEY ("organization_id") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "recipient" ADD CONSTRAINT "fk_recipient_bank_address" FOREIGN KEY ("bank_address_id") REFERENCES "bank_address"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "recipient" ADD CONSTRAINT "fk_recipient_mailing_address" FOREIGN KEY ("mailing_address_id") REFERENCES "address"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment" ADD CONSTRAINT "fk_payment_account" FOREIGN KEY ("account_id") REFERENCES "account"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment" ADD CONSTRAINT "fk_payment_recipient" FOREIGN KEY ("recipient_id") REFERENCES "recipient"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment" ADD CONSTRAINT "fk_payment_idempotency_key" FOREIGN KEY ("idempotency_key_id") REFERENCES "idempotency_key"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment" ADD CONSTRAINT "fk_payment_user" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "fk_transfer_account" FOREIGN KEY ("account_id") REFERENCES "account"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "fk_transfer_idempotency_key" FOREIGN KEY ("idempotency_key_id") REFERENCES "idempotency_key"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "fk_transfer_user" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
