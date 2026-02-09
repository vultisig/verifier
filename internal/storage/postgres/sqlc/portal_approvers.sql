-- Portal Approvers table queries

-- name: GetPortalApprover :one
SELECT * FROM portal_approvers WHERE public_key = $1;

-- name: IsStagingApprover :one
SELECT EXISTS(
  SELECT 1 FROM portal_approvers
  WHERE public_key = $1
    AND active = TRUE
    AND role IN ('staging_approver', 'listing_approver', 'admin')
) AS is_approver;

-- name: IsListingApprover :one
SELECT EXISTS(
  SELECT 1 FROM portal_approvers
  WHERE public_key = $1
    AND active = TRUE
    AND role IN ('listing_approver', 'admin')
) AS is_approver;

-- name: IsPortalAdmin :one
SELECT EXISTS(
  SELECT 1 FROM portal_approvers
  WHERE public_key = $1
    AND active = TRUE
    AND role = 'admin'
) AS is_admin;
