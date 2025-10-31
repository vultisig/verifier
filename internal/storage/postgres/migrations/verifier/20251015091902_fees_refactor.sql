-- +goose Up

DROP TRIGGER IF EXISTS prevent_fees_policy_deletion_with_active_fees ON public.plugin_policies;
DROP TRIGGER IF EXISTS trg_prevent_fees_update_if_policy_deleted ON public.fees;

DROP FUNCTION IF EXISTS public.check_active_fees_for_public_key();
DROP FUNCTION IF EXISTS public.prevent_fees_update_if_policy_deleted();

DROP VIEW IF EXISTS public.fees_view;
DROP VIEW IF EXISTS public.billing_periods;

DROP TABLE IF EXISTS public.fees;

-- +goose Down

CREATE TABLE IF NOT EXISTS public.fees (
                                           id uuid DEFAULT gen_random_uuid() NOT NULL,
    plugin_policy_billing_id uuid NOT NULL,
    transaction_id uuid,
    transaction_hash character varying(66),
    amount bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    charged_at date DEFAULT now() NOT NULL,
    collected_at timestamp without time zone
    );

ALTER TABLE public.fees ADD CONSTRAINT fees_pkey PRIMARY KEY (id);

CREATE INDEX IF NOT EXISTS idx_fees_billing_date ON public.fees USING btree (charged_at);
CREATE INDEX IF NOT EXISTS idx_fees_plugin_policy_billing_id ON public.fees USING btree (plugin_policy_billing_id);
CREATE INDEX IF NOT EXISTS idx_fees_transaction_id ON public.fees USING btree (transaction_id) WHERE (transaction_id IS NOT NULL);

ALTER TABLE public.fees
    ADD CONSTRAINT fk_billing FOREIGN KEY (plugin_policy_billing_id)
        REFERENCES public.plugin_policy_billing(id) ON DELETE CASCADE;

CREATE VIEW public.fees_view AS
SELECT pp.id AS policy_id,
       pp.plugin_id,
       ppb.id AS billing_id,
       pp.public_key,
       ppb.type,
       f.id,
       f.plugin_policy_billing_id,
       f.transaction_id,
       f.transaction_hash,
       f.amount,
       f.created_at,
       f.charged_at,
       f.collected_at
FROM public.plugin_policies pp
         JOIN public.plugin_policy_billing ppb ON ppb.plugin_policy_id = pp.id
         JOIN public.fees f ON f.plugin_policy_billing_id = ppb.id;

CREATE OR REPLACE VIEW public.billing_periods AS
SELECT pp.id AS plugin_policy_id,
       pp.active,
       ppb.id AS billing_id,
       ppb.frequency,
       ppb.amount,
       count(f.id) AS accrual_count,
       COALESCE(sum(f.amount), (0)::numeric) AS total_billed,
       COALESCE(max(f.charged_at), ppb.start_date) AS last_billed_date,
       (COALESCE(max(f.charged_at), ppb.start_date) +
        CASE ppb.frequency
            WHEN 'daily'::public.pricing_frequency THEN '1 day'::interval
                WHEN 'weekly'::public.pricing_frequency THEN '7 days'::interval
                WHEN 'biweekly'::public.pricing_frequency THEN '14 days'::interval
                WHEN 'monthly'::public.pricing_frequency THEN '1 mon'::interval
                ELSE NULL::interval
            END) AS next_billing_date
FROM public.plugin_policy_billing ppb
         JOIN public.plugin_policies pp ON ppb.plugin_policy_id = pp.id
         LEFT JOIN public.fees f ON f.plugin_policy_billing_id = ppb.id
WHERE ppb.type = 'recurring'::public.pricing_type
GROUP BY ppb.id, pp.id;

CREATE FUNCTION public.prevent_fees_update_if_policy_deleted() RETURNS trigger
    LANGUAGE plpgsql
AS $$
DECLARE
is_deleted boolean;
BEGIN
SELECT p.deleted INTO is_deleted
FROM public.plugin_policies p
         JOIN public.plugin_policy_billing b ON b.plugin_policy_id = p.id
WHERE b.id = COALESCE(NEW.plugin_policy_billing_id, OLD.plugin_policy_billing_id);
IF is_deleted THEN
        RAISE EXCEPTION 'Cannot modify fees for a deleted policy';
END IF;
RETURN NEW;
END;
$$;

CREATE TRIGGER trg_prevent_fees_update_if_policy_deleted
    BEFORE INSERT OR DELETE OR UPDATE ON public.fees
FOR EACH ROW EXECUTE FUNCTION public.prevent_fees_update_if_policy_deleted();

CREATE FUNCTION public.check_active_fees_for_public_key() RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    LOCK TABLE public.fees IN SHARE MODE;
    IF OLD.plugin_id = 'vultisig-fees-feee' THEN
        IF EXISTS (
            SELECT 1
            FROM public.fees_view fv
            WHERE fv.public_key = OLD.public_key
              AND fv.policy_id = OLD.id
        ) THEN
            RAISE EXCEPTION 'Cannot delete plugin policy: active fees exist for public key %', OLD.public_key;
END IF;
END IF;
RETURN OLD;
END;
$$;

CREATE TRIGGER prevent_fees_policy_deletion_with_active_fees
    BEFORE DELETE ON public.plugin_policies
    FOR EACH ROW EXECUTE FUNCTION public.check_active_fees_for_public_key();