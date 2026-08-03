package repository

import (
	"backend/model"
	"context"
)

func (r Repository) CreateOrUpdateNetConfig(ctx context.Context, config model.NetConfig) (model.NetConfig, error) {
	err := r.db.
		WithContext(ctx).
		Save(&config).
		Error
	return config, err
}

func (r Repository) GetNetConfig(ctx context.Context, id int) (model.NetConfig, error) {
	var member model.NetConfig
	err := r.db.
		WithContext(ctx).
		First(&member, id).
		Error
	return member, err
}

func (r Repository) ListNetConfigs(ctx context.Context, params model.NetworkRequestParams) ([]model.NetConfig, error) {
	var netConfigs []model.NetConfig
	tx := r.db.
		WithContext(ctx).
		Joins("LEFT JOIN members ON members.net_config_id = net_configs.id")

	tx = params.Apply(tx)
	err := tx.
		Order("Mac").
		Find(&netConfigs).
		Error
	return netConfigs, err
}
func (r Repository) DeleteNetConfig(ctx context.Context, id int) error {
	return r.db.
		WithContext(ctx).
		Delete(&model.NetConfig{}, id).
		Error
}

func (r Repository) Deactivate(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.
		WithContext(ctx).
		Table(netTable).
		Where("id IN ?", ids).
		Updates(map[string]any{"disabled": true}).
		Error
}

func (r Repository) GetAllIPs(ctx context.Context) ([]string, error) {
	var ips []string
	err := r.db.
		WithContext(ctx).
		Table(netTable).
		Select("ip").
		Order("ip").
		Find(&ips).
		Error
	return ips, err
}
