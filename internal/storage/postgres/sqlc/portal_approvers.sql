-- Portal Approvers table queries

-- name: GetPortalApprover :one
SELECT public_key, active, added_via, added_by_public_key, created_at, updated_at
FROM portal_approvers WHERE public_key = $1;

-- name: IsStagingApprover :one
SELECT EXISTS(
  SELECT 1 FROM portal_approvers
  WHERE public_key = $1
    AND active = TRUE
) AS is_approver;

-- name: IsListingApprover :one
SELECT EXISTS(
  SELECT 1 FROM portal_approvers
  WHERE public_key = $1
    AND active = TRUE
) AS is_approver;

-- name: IsPortalAdmin :one
SELECT EXISTS(
  SELECT 1 FROM portal_approvers
  WHERE public_key = $1
    AND active = TRUE
) AS is_admin;
