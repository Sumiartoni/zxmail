package postal

import corepostal "zxmail/backend/internal/postal"

type Service struct {
	client *corepostal.Client
}

func NewService(client *corepostal.Client) *Service {
	return &Service{client: client}
}
