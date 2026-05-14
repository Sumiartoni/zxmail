# Usage and Quota

Usage is derived from Postal webhook events.

Current rules:
- accepted events increment credential counters and usage records
- delivered, bounced, deferred, and rejected events are tracked for reporting
- Redis is used for fast advisory counters
- PostgreSQL remains the source of truth

Important limitation:
- customers still send directly to Postal
- without a future SMTP gateway, pre-send enforcement remains limited

Admin controls:
- reset organization usage
- limit or unlimit credentials
- update organization retention and quota policy
