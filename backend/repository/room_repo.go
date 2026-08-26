package repository

import (
	"backend/model"
	"context"
)

const netTable = "net_configs"

func (r Repository) ListRooms(ctx context.Context, params model.RoomRequestParams) ([]model.Room, error) {
	var rooms []model.Room

	tx := r.db.
		WithContext(ctx).
		Joins("Member").
		Joins("Member.NetConfig")
	tx = params.Apply(tx)
	err := tx.
		Find(&rooms).
		Error
	return rooms, err
}

func (r Repository) GetRoom(ctx context.Context, number string) (model.Room, error) {
	var room model.Room
	err := r.db.
		WithContext(ctx).
		First(&room, "number = ?", number).
		Error
	return room, err
}
