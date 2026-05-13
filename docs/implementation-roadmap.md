# Implementation Roadmap

## Phase order
1. Auth and organization membership.
2. Domain onboarding and DNS verification.
3. SMTP credential issuance with encryption and show-once password flow.
4. Postal integration layer and webhook persistence.
5. Send logs, bounces, suppressions, and quota enforcement.
6. Admin and customer dashboard data wiring.
7. Deployment hardening, monitoring, and backups.

## Immediate next milestones
- Replace scaffold auth middleware with JWT validation and role resolution.
- Add repository and service layers behind the current route placeholders.
- Wire Redis jobs for DNS verification polling and quota windows.
- Implement Postal API client methods needed for server, credential, and webhook operations.
- Connect dashboard pages to backend endpoints with session-aware data fetching.
