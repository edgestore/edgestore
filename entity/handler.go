package entity

import (
	"context"

	"github.com/edgestore/edgestore/internal/model"
)

func NewSetEntityToCacheHandler(entity *Entity, svc *Service) func() error {
	return func() error {
		if err := svc.setEntityToCache(context.Background(), entity); err != nil {
			return err
		}

		return nil
	}
}

func NewApplyEntityHandler(cmd model.Command, svc *Service) func() error {
	return func() error {
		ctx := context.Background()
		entity, err := svc.applyEntityToDatabase(ctx, cmd)
		if err != nil {
			return err
		}

		// Set aside cache
		if err := svc.setEntityToCache(ctx, entity); err != nil {
			return err
		}

		return nil
	}
}
