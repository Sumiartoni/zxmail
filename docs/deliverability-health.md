# Deliverability Health

zxMail v2 surfaces health indicators, not inbox guarantees.

Signals used:
- SPF
- DKIM
- DMARC
- bounce rate
- deferred rate
- rejection rate
- quota-limited state

Alerts are generated when:
- bounce rate is high
- deferred or rejected activity is elevated
- domain health score drops
