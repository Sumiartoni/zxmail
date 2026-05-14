# Deployment

Compose services:
- `postgres`
- `redis`
- `backend`
- `worker`
- `frontend`
- `caddy`

Checklist:
1. set `DASHBOARD_HOST=app.zxmail.site`
2. set `API_HOST=api.zxmail.site`
3. keep `smtp.zxmail.site` as `DNS only` in Cloudflare
4. ensure PTR/rDNS points to `smtp.zxmail.site`
5. apply `000001_init` and `000002_production_v2` migrations
6. start the stack with `docker compose up --build -d`
7. configure Postal webhook secret and callback URL
8. verify `/health`, `/ready`, and worker `/health`
9. verify `COOKIE_DOMAIN`, `CORS_ALLOW_ORIGINS`, and `FRONTEND_ORIGIN` match the public hosts
10. verify `ENCRYPTION_KEYS` and `ACTIVE_ENCRYPTION_KEY_ID` are identical on every backend deploy

Backup example:
- `docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" > backups/zxmail-$(date +%F).sql`

Restore note:
- restore into a staging database first, then use `psql` or `docker compose exec -T postgres psql ...` for production recovery only after verification

Do not expose:
- PostgreSQL publicly
- Redis publicly
