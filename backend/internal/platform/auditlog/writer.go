package auditlog

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type ExecQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func Insert(
	ctx context.Context,
	querier ExecQuerier,
	actorUserID *uuid.UUID,
	organizationID *uuid.UUID,
	action string,
	targetType string,
	targetID *uuid.UUID,
	metadata map[string]any,
) error {
	payload, err := MarshalAuditDetails(metadata)
	if err != nil {
		return err
	}

	_, err = querier.Exec(
		ctx,
		`INSERT INTO audit_logs (actor_user_id, organization_id, action, target_type, target_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
		actorUserID,
		organizationID,
		action,
		targetType,
		targetID,
		payload,
	)
	return err
}
