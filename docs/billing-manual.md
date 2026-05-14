# Manual Billing

Supported payment providers in v2:
- `manual_bank_transfer`
- `manual_qris`

Operational flow:
1. Admin publishes a plan.
2. Admin assigns a subscription to an organization.
3. zxMail issues an invoice and pending payment record if the plan is billable.
4. Customer reviews billing status and invoice list in `/billing`.
5. Admin approves or rejects the payment in `/admin/payments`.

Notes:
- customer self-service plan changes are not enabled in v2
- no Stripe dependency exists
- payment metadata must stay sanitized in audit logs
