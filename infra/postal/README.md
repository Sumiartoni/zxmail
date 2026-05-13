# External Postal Setup Notes

zxMail Production v1 uses Postal as the SMTP core, but this repository's main `docker-compose.yml` intentionally does not run a full Postal cluster.

Reason:
- Postal needs dedicated SMTP networking and deliverability-sensitive setup.
- Production v1 should keep the dashboard/API stack simple while allowing Postal to be deployed on the same VPS or a separate dedicated host with operator-managed mail infrastructure.

Use `infra/postal/postal.example.yml` as a configuration template when you deploy Postal separately.

Minimum operator checklist:
- Publish `smtp.zxmail.site` as a DNS-only record in Cloudflare. Never proxy the SMTP hostname.
- Configure the VPS provider PTR/rDNS record so it points to `smtp.zxmail.site`.
- Open and route SMTP ports `25`, `465`, and `587` to the Postal host.
- Point `POSTAL_BASE_URL` in zxMail to the Postal web/API base URL.
- Set Postal to send webhooks to `https://dashboard.zxmail.site/webhooks/postal/event`.
- Reuse the same `POSTAL_WEBHOOK_SECRET` value in both Postal and zxMail.
