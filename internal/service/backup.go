package service

import (
	"ariand/internal/backup"
	"ariand/internal/db/sqlc"
	"context"

	"github.com/google/uuid"
)

type BackupService interface {
	ExportAll(ctx context.Context, userID uuid.UUID) (*backup.Backup, error)
	ImportAll(ctx context.Context, userID uuid.UUID, data *backup.Backup) error
}

type backupSvc struct {
	db *sqlc.Queries
}

func newBackupSvc(db *sqlc.Queries) BackupService {
	return &backupSvc{db: db}
}

func (s *backupSvc) ExportAll(ctx context.Context, userID uuid.UUID) (*backup.Backup, error) {
	return backup.ExportAll(ctx, s.db, userID)
}

func (s *backupSvc) ImportAll(ctx context.Context, userID uuid.UUID, data *backup.Backup) error {
	return backup.ImportAll(ctx, s.db, userID, data)
}
