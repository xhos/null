package api

import (
	"context"

	pb "null/internal/gen/null/v1"

	"connectrpc.com/connect"
)

func (s *Server) ExportBackup(ctx context.Context, req *connect.Request[pb.ExportBackupRequest]) (*connect.Response[pb.ExportBackupResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	backup, err := s.services.Backup.ExportAll(ctx, userID)
	if err != nil {
		return nil, wrapErr(err)
	}

	protoBackup := s.services.Backup.BackupToProto(backup)

	return connect.NewResponse(&pb.ExportBackupResponse{
		Backup: protoBackup,
	}), nil
}

func (s *Server) ImportBackup(ctx context.Context, req *connect.Request[pb.ImportBackupRequest]) (*connect.Response[pb.ImportBackupResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	backup := s.services.Backup.BackupFromProto(req.Msg.Backup)

	err = s.services.Backup.ImportAll(ctx, userID, backup)
	if err != nil {
		return nil, wrapErr(err)
	}

	return connect.NewResponse(&pb.ImportBackupResponse{
		CategoriesImported:   int32(len(backup.Categories)),
		AccountsImported:     int32(len(backup.Accounts)),
		TransactionsImported: int32(len(backup.Transactions)),
		RulesImported:        int32(len(backup.Rules)),
	}), nil
}
