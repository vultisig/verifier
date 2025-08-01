
CREATE TYPE "billing_asset" AS ENUM (
    'usdc'
);

CREATE TYPE "plugin_category" AS ENUM (
    'ai-agent',
    'plugin'
);

CREATE TYPE "plugin_id" AS ENUM (
    'vultisig-dca-0000',
    'vultisig-payroll-0000',
    'vultisig-fees-feee'
);

CREATE TYPE "pricing_asset" AS ENUM (
    'usdc'
);

CREATE TYPE "pricing_frequency" AS ENUM (
    'daily',
    'weekly',
    'biweekly',
    'monthly'
);

CREATE TYPE "pricing_metric" AS ENUM (
    'fixed'
);

CREATE TYPE "pricing_type" AS ENUM (
    'once',
    'recurring',
    'per-tx'
);

CREATE TYPE "tx_indexer_status" AS ENUM (
    'PROPOSED',
    'VERIFIED',
    'SIGNED'
);

CREATE TYPE "tx_indexer_status_onchain" AS ENUM (
    'PENDING',
    'SUCCESS',
    'FAIL'
);

CREATE FUNCTION "check_active_fees_for_public_key"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
BEGIN
    -- Lowered lock level to SHARE MODE to reduce deadlock risk with concurrent INSERTs.
    -- This should be sufficient to prevent concurrent modifications for our check.
    LOCK TABLE fees IN SHARE MODE;
    -- If we're deleting a vultisig-fees-feee policy
    IF OLD.plugin_id = 'vultisig-fees-feee' THEN
        -- Check if there are any active fees for this public key
        IF EXISTS (
            SELECT 1 
            FROM fees_view fv 
            WHERE fv.public_key = OLD.public_key 
            AND fv.policy_id = OLD.id
        ) THEN
            RAISE EXCEPTION 'Cannot delete plugin policy: active fees exist for public key %', OLD.public_key;
        END IF;
    END IF;
    
    RETURN OLD;
END;
$$;

CREATE FUNCTION "prevent_billing_update_if_policy_deleted"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT deleted INTO is_deleted FROM plugin_policies WHERE id = COALESCE(NEW.plugin_policy_id, OLD.plugin_policy_id);
    IF is_deleted THEN
        RAISE EXCEPTION 'Cannot modify billing for a deleted policy';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION "prevent_fees_update_if_policy_deleted"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT p.deleted INTO is_deleted
    FROM plugin_policies p
    JOIN plugin_policy_billing b ON b.plugin_policy_id = p.id
    WHERE b.id = COALESCE(NEW.plugin_policy_billing_id, OLD.plugin_policy_billing_id);
    IF is_deleted THEN
        RAISE EXCEPTION 'Cannot modify fees for a deleted policy';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION "prevent_insert_if_policy_deleted"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
BEGIN
    IF NEW.deleted = true THEN
        RAISE EXCEPTION 'Cannot insert a deleted policy';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION "prevent_tx_indexer_update_if_policy_deleted"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT deleted INTO is_deleted FROM plugin_policies WHERE id = COALESCE(NEW.policy_id, OLD.policy_id);
    IF is_deleted THEN
        RAISE EXCEPTION 'Cannot modify tx_indexer for a deleted policy';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION "prevent_update_if_policy_deleted"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
BEGIN
    IF OLD.deleted = true THEN
        RAISE EXCEPTION 'Cannot update a deleted policy';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION "set_policy_inactive_on_delete"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
BEGIN
    IF NEW.deleted = true THEN
        NEW.active := false;
    END IF;
    RETURN NEW;
END;
$$;

CREATE VIEW "billing_periods" AS
SELECT
    NULL::"uuid" AS "plugin_policy_id",
    NULL::boolean AS "active",
    NULL::"uuid" AS "billing_id",
    NULL::"pricing_frequency" AS "frequency",
    NULL::bigint AS "amount",
    NULL::bigint AS "accrual_count",
    NULL::numeric AS "total_billed",
    NULL::"date" AS "last_billed_date",
    NULL::timestamp without time zone AS "next_billing_date";

CREATE TABLE "fees" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "plugin_policy_billing_id" "uuid" NOT NULL,
    "transaction_id" "uuid",
    "transaction_hash" character varying(66),
    "amount" bigint NOT NULL,
    "created_at" timestamp without time zone DEFAULT "now"() NOT NULL,
    "charged_at" "date" DEFAULT "now"() NOT NULL,
    "collected_at" timestamp without time zone
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
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "deleted" boolean DEFAULT false NOT NULL
);

CREATE TABLE "plugin_policy_billing" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "type" "pricing_type" NOT NULL,
    "frequency" "pricing_frequency",
    "start_date" "date" DEFAULT CURRENT_DATE NOT NULL,
    "amount" bigint NOT NULL,
    "asset" "pricing_asset" NOT NULL,
    "plugin_policy_id" "uuid" NOT NULL,
    CONSTRAINT "frequency_check" CHECK (((("type" = 'recurring'::"pricing_type") AND ("frequency" IS NOT NULL)) OR (("type" = ANY (ARRAY['per-tx'::"public"."pricing_type", 'once'::"public"."pricing_type"])) AND ("frequency" IS NULL))))
);

CREATE VIEW "fees_view" AS
 SELECT "pp"."id" AS "policy_id",
    "pp"."plugin_id",
    "ppb"."id" AS "billing_id",
    "pp"."public_key",
    "ppb"."type",
    "f"."id",
    "f"."plugin_policy_billing_id",
    "f"."transaction_id",
    "f"."transaction_hash",
    "f"."amount",
    "f"."created_at",
    "f"."charged_at",
    "f"."collected_at"
   FROM (("plugin_policies" "pp"
     JOIN "plugin_policy_billing" "ppb" ON (("ppb"."plugin_policy_id" = "pp"."id")))
     JOIN "fees" "f" ON (("f"."plugin_policy_billing_id" = "ppb"."id")));

CREATE TABLE "plugin_apikey" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "plugin_id" "plugin_id" NOT NULL,
    "apikey" "text" NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "expires_at" timestamp with time zone,
    "status" integer DEFAULT 1 NOT NULL,
    CONSTRAINT "plugin_apikey_status_check" CHECK (("status" = ANY (ARRAY[0, 1])))
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
    "category" "plugin_category" NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL
);

CREATE TABLE "pricings" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "type" "pricing_type" NOT NULL,
    "frequency" "pricing_frequency",
    "amount" bigint NOT NULL,
    "asset" "pricing_asset" NOT NULL,
    "metric" "pricing_metric" NOT NULL,
    "plugin_id" "plugin_id" NOT NULL,
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT "frequency_check" CHECK (((("type" = 'recurring'::"pricing_type") AND ("frequency" IS NOT NULL)) OR (("type" = ANY (ARRAY['per-tx'::"public"."pricing_type", 'once'::"public"."pricing_type"])) AND ("frequency" IS NULL))))
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

CREATE TABLE "tx_indexer" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "plugin_id" character varying(255) NOT NULL,
    "tx_hash" character varying(255),
    "chain_id" integer NOT NULL,
    "policy_id" "uuid" NOT NULL,
    "token_id" character varying(255) NOT NULL,
    "from_public_key" character varying(255) NOT NULL,
    "to_public_key" character varying(255) NOT NULL,
    "proposed_tx_hex" "text" NOT NULL,
    "status" "tx_indexer_status" DEFAULT 'PROPOSED'::"public"."tx_indexer_status" NOT NULL,
    "status_onchain" "tx_indexer_status_onchain",
    "lost" boolean DEFAULT false NOT NULL,
    "broadcasted_at" timestamp without time zone,
    "created_at" timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
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

ALTER TABLE ONLY "fees"
    ADD CONSTRAINT "fees_pkey" PRIMARY KEY ("id");

ALTER TABLE ONLY "plugin_apikey"
    ADD CONSTRAINT "plugin_apikey_apikey_key" UNIQUE ("apikey");

ALTER TABLE ONLY "plugin_apikey"
    ADD CONSTRAINT "plugin_apikey_pkey" PRIMARY KEY ("id");

ALTER TABLE ONLY "plugin_policies"
    ADD CONSTRAINT "plugin_policies_pkey" PRIMARY KEY ("id");

ALTER TABLE ONLY "plugin_policy_billing"
    ADD CONSTRAINT "plugin_policy_billing_pkey" PRIMARY KEY ("id");

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

ALTER TABLE ONLY "tx_indexer"
    ADD CONSTRAINT "tx_indexer_pkey" PRIMARY KEY ("id");

ALTER TABLE ONLY "vault_tokens"
    ADD CONSTRAINT "vault_tokens_pkey" PRIMARY KEY ("id");

ALTER TABLE ONLY "vault_tokens"
    ADD CONSTRAINT "vault_tokens_token_id_key" UNIQUE ("token_id");

CREATE INDEX "idx_fees_billing_date" ON "fees" USING "btree" ("charged_at");

CREATE INDEX "idx_fees_plugin_policy_billing_id" ON "fees" USING "btree" ("plugin_policy_billing_id");

CREATE INDEX "idx_fees_transaction_id" ON "fees" USING "btree" ("transaction_id") WHERE ("transaction_id" IS NOT NULL);

CREATE INDEX "idx_plugin_apikey_apikey" ON "plugin_apikey" USING "btree" ("apikey");

CREATE INDEX "idx_plugin_apikey_plugin_id" ON "plugin_apikey" USING "btree" ("plugin_id");

CREATE INDEX "idx_plugin_policies_active" ON "plugin_policies" USING "btree" ("active");

CREATE INDEX "idx_plugin_policies_plugin_id" ON "plugin_policies" USING "btree" ("plugin_id");

CREATE INDEX "idx_plugin_policies_public_key" ON "plugin_policies" USING "btree" ("public_key");

CREATE INDEX "idx_plugin_policy_billing_id" ON "plugin_policy_billing" USING "btree" ("id");

CREATE INDEX "idx_plugin_policy_sync_policy_id" ON "plugin_policy_sync" USING "btree" ("policy_id");

CREATE INDEX "idx_reviews_plugin_id" ON "reviews" USING "btree" ("plugin_id");

CREATE INDEX "idx_reviews_public_key" ON "reviews" USING "btree" ("public_key");

CREATE INDEX "idx_tx_indexer_key" ON "tx_indexer" USING "btree" ("chain_id", "plugin_id", "policy_id", "token_id", "to_public_key", "created_at");

CREATE INDEX "idx_tx_indexer_policy_id_created_at" ON "tx_indexer" USING "btree" ("policy_id", "created_at");

CREATE INDEX "idx_tx_indexer_status_onchain_lost" ON "tx_indexer" USING "btree" ("status_onchain", "lost");

CREATE INDEX "idx_vault_tokens_public_key" ON "vault_tokens" USING "btree" ("public_key");

CREATE INDEX "idx_vault_tokens_token_id" ON "vault_tokens" USING "btree" ("token_id");

CREATE UNIQUE INDEX "unique_fees_policy_per_public_key" ON "plugin_policies" USING "btree" ("plugin_id", "public_key") WHERE (("plugin_id" = 'vultisig-fees-feee'::"public"."plugin_id") AND ("active" = true));

CREATE OR REPLACE VIEW "billing_periods" AS
 SELECT "pp"."id" AS "plugin_policy_id",
    "pp"."active",
    "ppb"."id" AS "billing_id",
    "ppb"."frequency",
    "ppb"."amount",
    "count"("f"."id") AS "accrual_count",
    COALESCE("sum"("f"."amount"), (0)::numeric) AS "total_billed",
    COALESCE("max"("f"."charged_at"), "ppb"."start_date") AS "last_billed_date",
    (COALESCE("max"("f"."charged_at"), "ppb"."start_date") +
        CASE "ppb"."frequency"
            WHEN 'daily'::"pricing_frequency" THEN '1 day'::interval
            WHEN 'weekly'::"pricing_frequency" THEN '7 days'::interval
            WHEN 'biweekly'::"pricing_frequency" THEN '14 days'::interval
            WHEN 'monthly'::"pricing_frequency" THEN '1 mon'::interval
            ELSE NULL::interval
        END) AS "next_billing_date"
   FROM (("plugin_policy_billing" "ppb"
     JOIN "plugin_policies" "pp" ON (("ppb"."plugin_policy_id" = "pp"."id")))
     LEFT JOIN "fees" "f" ON (("f"."plugin_policy_billing_id" = "ppb"."id")))
  WHERE ("ppb"."type" = 'recurring'::"pricing_type")
  GROUP BY "ppb"."id", "pp"."id";

CREATE TRIGGER "prevent_fees_policy_deletion_with_active_fees" BEFORE DELETE ON "plugin_policies" FOR EACH ROW EXECUTE FUNCTION "public"."check_active_fees_for_public_key"();

CREATE TRIGGER "trg_prevent_billing_update_if_policy_deleted" BEFORE INSERT OR DELETE OR UPDATE ON "plugin_policy_billing" FOR EACH ROW EXECUTE FUNCTION "public"."prevent_billing_update_if_policy_deleted"();

CREATE TRIGGER "trg_prevent_fees_update_if_policy_deleted" BEFORE INSERT OR DELETE OR UPDATE ON "fees" FOR EACH ROW EXECUTE FUNCTION "public"."prevent_fees_update_if_policy_deleted"();

CREATE TRIGGER "trg_prevent_insert_if_policy_deleted" BEFORE INSERT ON "plugin_policies" FOR EACH ROW EXECUTE FUNCTION "public"."prevent_insert_if_policy_deleted"();

CREATE TRIGGER "trg_prevent_tx_indexer_update_if_policy_deleted" BEFORE INSERT OR DELETE OR UPDATE ON "tx_indexer" FOR EACH ROW EXECUTE FUNCTION "public"."prevent_tx_indexer_update_if_policy_deleted"();

CREATE TRIGGER "trg_prevent_update_if_policy_deleted" BEFORE UPDATE ON "plugin_policies" FOR EACH ROW WHEN (("old"."deleted" = true)) EXECUTE FUNCTION "public"."prevent_update_if_policy_deleted"();

CREATE TRIGGER "trg_set_policy_inactive_on_delete" BEFORE INSERT OR UPDATE ON "plugin_policies" FOR EACH ROW WHEN (("new"."deleted" = true)) EXECUTE FUNCTION "public"."set_policy_inactive_on_delete"();

ALTER TABLE ONLY "fees"
    ADD CONSTRAINT "fk_billing" FOREIGN KEY ("plugin_policy_billing_id") REFERENCES "plugin_policy_billing"("id") ON DELETE CASCADE;

ALTER TABLE ONLY "plugin_policy_billing"
    ADD CONSTRAINT "fk_plugin_policy" FOREIGN KEY ("plugin_policy_id") REFERENCES "plugin_policies"("id") ON DELETE CASCADE;

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

ALTER TABLE ONLY "pricings"
    ADD CONSTRAINT "pricings_plugin_id_fkey" FOREIGN KEY ("plugin_id") REFERENCES "plugins"("id") ON DELETE CASCADE;

ALTER TABLE ONLY "reviews"
    ADD CONSTRAINT "reviews_plugin_id_fkey" FOREIGN KEY ("plugin_id") REFERENCES "plugins"("id") ON DELETE CASCADE;

