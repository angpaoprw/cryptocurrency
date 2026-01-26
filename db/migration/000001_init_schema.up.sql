CREATE TABLE "wallets" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "blockchain" text NOT NULL,
  "address" text NOT NULL,
  "public_key" text NOT NULL,
  "private_key" text NOT NULL,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now())
);



