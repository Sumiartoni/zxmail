# Security

Current security posture:
- HttpOnly cookie auth for browser sessions
- JWT validation with issuer and algorithm checks
- Redis-backed login throttling and temporary lockout
- strict CORS in production
- HMAC verification for Postal webhooks
- SMTP credential encryption with multi-key rotation structure
- raw event sanitization for customer logs
- audit log sanitization for secrets and tokens

Manual review items:
- production secret rotation process
- Postal TLS and origin hardening
- backup and restore procedure
- browser auth migration to split app and API domains in production
