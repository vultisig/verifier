CREATE TYPE "plugin_category" AS ENUM (
    'ai-agent',
    'plugin'
);
CREATE TYPE "plugin_id" AS ENUM (
    'vultisig-dca-0000',
    'vultisig-payroll-0000',
    'vultisig-fees-feee'
);
CREATE TYPE "pricing_frequency" AS ENUM (
    'annual',
    'monthly',
    'weekly'
);
CREATE TYPE "pricing_metric" AS ENUM (
    'fixed',
    'percentage'
);
CREATE TYPE "pricing_type" AS ENUM (
    'free',
    'single',
    'recurring',
    'per-tx'
);
CREATE FUNCTION "trigger_set_timestamp"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$;
CREATE TABLE "plugin_apikey" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "plugin_id" "plugin_id" NOT NULL,
    "apikey" "text" NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "expires_at" timestamp with time zone,
    "status" integer DEFAULT 1 NOT NULL,
    CONSTRAINT "plugin_apikey_status_check" CHECK (("status" = ANY (ARRAY[0, 1])))
);
CREATE TABLE "plugin_policies" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "public_key" "text" NOT NULL,
    "plugin_id" "plugin_id" NOT NULL,
    "plugin_version" "text" NOT NULL,
    "policy_version" integer NOT NULL,
    "signature" "text" NOT NULL,
    "recipe" "text" NOT NULL,
    "active" boolean DEFAULT true NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL
);
CREATE TABLE "plugin_policy_sync" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "policy_id" "uuid" NOT NULL,
    "plugin_id" "plugin_id" NOT NULL,
    "sync_type" integer NOT NULL,
    "signature" "text",
    "status" integer DEFAULT 0 NOT NULL,
    "reason" "text",
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL
);
CREATE TABLE "plugin_ratings" (
    "plugin_id" "plugin_id" NOT NULL,
    "avg_rating" numeric(3,2) DEFAULT 0 NOT NULL,
    "total_ratings" integer DEFAULT 0 NOT NULL,
    "rating_1_count" integer DEFAULT 0 NOT NULL,
    "rating_2_count" integer DEFAULT 0 NOT NULL,
    "rating_3_count" integer DEFAULT 0 NOT NULL,
    "rating_4_count" integer DEFAULT 0 NOT NULL,
    "rating_5_count" integer DEFAULT 0 NOT NULL,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL
);
CREATE TABLE "plugin_tags" (
    "plugin_id" "plugin_id" NOT NULL,
    "tag_id" "uuid" NOT NULL
);
CREATE TABLE "plugins" (
    "id" "plugin_id" NOT NULL,
    "title" character varying(255) NOT NULL,
    "description" "text" DEFAULT ''::"text" NOT NULL,
    "server_endpoint" "text" NOT NULL,
    "pricing_id" "uuid",
    "category" "plugin_category" NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL
);
CREATE TABLE "pricings" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "type" "pricing_type" NOT NULL,
    "frequency" "pricing_frequency",
    "amount" numeric(10,2) NOT NULL,
    "metric" "pricing_metric" NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL
);
CREATE TABLE "reviews" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "plugin_id" "plugin_id",
    "public_key" "text" NOT NULL,
    "rating" integer NOT NULL,
    "comment" "text",
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT "reviews_rating_check" CHECK ((("rating" >= 1) AND ("rating" <= 5)))
);
CREATE TABLE "tags" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "name" character varying(100) NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL
);
CREATE TABLE "vault_tokens" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "token_id" character varying(255) NOT NULL,
    "public_key" character varying(255) NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "expires_at" timestamp with time zone NOT NULL,
    "last_used_at" timestamp with time zone,
    "revoked_at" timestamp with time zone,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL
);
ALTER TABLE ONLY "plugin_apikey"
    ADD CONSTRAINT "plugin_apikey_apikey_key" UNIQUE ("apikey");
ALTER TABLE ONLY "plugin_apikey"
    ADD CONSTRAINT "plugin_apikey_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "plugin_policies"
    ADD CONSTRAINT "plugin_policies_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "plugin_policy_sync"
    ADD CONSTRAINT "plugin_policy_sync_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "plugin_ratings"
    ADD CONSTRAINT "plugin_ratings_pkey" PRIMARY KEY ("plugin_id");
ALTER TABLE ONLY "plugin_tags"
    ADD CONSTRAINT "plugin_tags_pkey" PRIMARY KEY ("plugin_id", "tag_id");
ALTER TABLE ONLY "plugins"
    ADD CONSTRAINT "plugins_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "pricings"
    ADD CONSTRAINT "pricings_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "reviews"
    ADD CONSTRAINT "reviews_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "tags"
    ADD CONSTRAINT "tags_name_key" UNIQUE ("name");
ALTER TABLE ONLY "tags"
    ADD CONSTRAINT "tags_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "vault_tokens"
    ADD CONSTRAINT "vault_tokens_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "vault_tokens"
    ADD CONSTRAINT "vault_tokens_token_id_key" UNIQUE ("token_id");
CREATE INDEX "idx_plugin_apikey_apikey" ON "plugin_apikey" USING "btree" ("apikey");
CREATE INDEX "idx_plugin_apikey_plugin_id" ON "plugin_apikey" USING "btree" ("plugin_id");
CREATE INDEX "idx_plugin_policies_active" ON "plugin_policies" USING "btree" ("active");
CREATE INDEX "idx_plugin_policies_plugin_id" ON "plugin_policies" USING "btree" ("plugin_id");
CREATE INDEX "idx_plugin_policies_public_key" ON "plugin_policies" USING "btree" ("public_key");
CREATE INDEX "idx_plugin_policy_sync_policy_id" ON "plugin_policy_sync" USING "btree" ("policy_id");
CREATE INDEX "idx_reviews_plugin_id" ON "reviews" USING "btree" ("plugin_id");
CREATE INDEX "idx_reviews_public_key" ON "reviews" USING "btree" ("public_key");
CREATE INDEX "idx_vault_tokens_public_key" ON "vault_tokens" USING "btree" ("public_key");
CREATE INDEX "idx_vault_tokens_token_id" ON "vault_tokens" USING "btree" ("token_id");
ALTER TABLE ONLY "plugin_apikey"
    ADD CONSTRAINT "plugin_apikey_plugin_id_fkey" FOREIGN KEY ("plugin_id") REFERENCES "plugins"("id") ON DELETE CASCADE;
ALTER TABLE ONLY "plugin_policy_sync"
    ADD CONSTRAINT "plugin_policy_sync_plugin_id_fkey" FOREIGN KEY ("plugin_id") REFERENCES "plugins"("id") ON DELETE CASCADE;
ALTER TABLE ONLY "plugin_policy_sync"
    ADD CONSTRAINT "plugin_policy_sync_policy_id_fkey" FOREIGN KEY ("policy_id") REFERENCES "plugin_policies"("id") ON DELETE CASCADE;
ALTER TABLE ONLY "plugin_ratings"
    ADD CONSTRAINT "plugin_ratings_plugin_id_fkey" FOREIGN KEY ("plugin_id") REFERENCES "plugins"("id") ON DELETE CASCADE;
ALTER TABLE ONLY "plugin_tags"
    ADD CONSTRAINT "plugin_tags_plugin_id_fkey" FOREIGN KEY ("plugin_id") REFERENCES "plugins"("id") ON DELETE CASCADE;
ALTER TABLE ONLY "plugin_tags"
    ADD CONSTRAINT "plugin_tags_tag_id_fkey" FOREIGN KEY ("tag_id") REFERENCES "tags"("id") ON DELETE CASCADE;
ALTER TABLE ONLY "plugins"
    ADD CONSTRAINT "plugins_pricing_id_fkey" FOREIGN KEY ("pricing_id") REFERENCES "pricings"("id");
ALTER TABLE ONLY "reviews"
    ADD CONSTRAINT "reviews_plugin_id_fkey" FOREIGN KEY ("plugin_id") REFERENCES "plugins"("id") ON DELETE CASCADE;
