package association

import (
	"context"

	"github.com/edgestore/edgestore/internal/model"
)

func NewSetAssociationToCacheHandler(assoc *Association, svc *Service) func() error {
	return func() error {
		if err := svc.setAssociationToCache(context.Background(), assoc); err != nil {
			return err
		}

		return nil
	}
}

func NewApplyAssociationHandler(cmd model.Command, svc *Service) func() error {
	return func() error {
		ctx := context.Background()
		assoc, err := svc.applyAssociationToDatabase(ctx, cmd)
		if err != nil {
			return err
		}

		// Set aside cache
		if err := svc.setAssociationToCache(ctx, assoc); err != nil {
			return err
		}

		return nil
	}
}
