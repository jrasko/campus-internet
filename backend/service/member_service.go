package service

import (
	"backend/model"
	"backend/service/oui"
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func (s *Service) CreateOrUpdateMember(ctx context.Context, member model.Member) (model.Member, error) {
	err := s.validate.Struct(member)
	if err != nil {
		return model.Member{}, mapValidationError(err)
	}

	// test if room exists
	_, err = s.roomRepo.GetRoom(ctx, member.RoomNr)
	if err != nil {
		return model.Member{}, model.WrapGormError(err)
	}

	member.Sanitize()
	member.LastEditor, _ = ctx.Value(model.FieldUsername).(string)
	if len(member.NetConfig.Mac) >= 8 {
		member.NetConfig.Manufacturer = oui.Mappings[member.NetConfig.Mac[:8]]
	}

	if member.NetConfig.IP == "" {
		member.NetConfig.IP, err = s.ipService.GetUnusedIP(ctx)
		if err != nil {
			return model.Member{}, err
		}
	}
	// save net config
	config, err := s.netRepo.CreateOrUpdateNetConfig(ctx, member.NetConfig)
	if err != nil {
		return model.Member{}, model.WrapGormError(err)
	}
	member.NetConfig = config
	member.NetConfigID = config.ID
	// save member
	member, err = s.memberRepo.CreateOrUpdateMember(ctx, specialize(member))
	if err != nil {
		return model.Member{}, model.WrapGormError(err)
	}

	err = s.UpdateDhcpFile(ctx)
	if err != nil {
		return model.Member{}, err
	}

	return member, err
}

func (s *Service) GetMember(ctx context.Context, id int) (model.Member, error) {
	member, err := s.memberRepo.GetMember(ctx, id)
	if err != nil {
		return model.Member{}, model.WrapGormError(err)
	}
	return member, nil
}

func (s *Service) DeleteMember(ctx context.Context, id int) error {
	member, err := s.GetMember(ctx, id)
	if err != nil {
		return err
	}

	err = s.memberRepo.DeleteMember(ctx, id)
	if err != nil {
		return model.WrapGormError(err)
	}

	err = s.netRepo.DeleteNetConfig(ctx, member.NetConfigID)
	if err != nil {
		return model.WrapGormError(fmt.Errorf("deleting net config, %w", err))
	}

	return s.UpdateDhcpFile(ctx)
}

func (s *Service) ResetPayment(ctx context.Context) error {
	return model.WrapGormError(s.memberRepo.ResetPayment(ctx))
}

func (s *Service) TogglePayment(ctx context.Context, id int) error {
	config, err := s.memberRepo.GetMember(ctx, id)
	if err != nil {
		return model.WrapGormError(err)
	}
	config.HasPaid = !config.HasPaid
	_, err = s.memberRepo.CreateOrUpdateMember(ctx, config)
	return model.WrapGormError(err)
}

func (s *Service) Punish(ctx context.Context) error {
	hasPaid := false
	isOccupied := true

	nonPayers, err := s.roomRepo.ListRooms(ctx, model.RoomRequestParams{HasPaid: &hasPaid, IsOccupied: &isOccupied})
	if err != nil {
		return model.WrapGormError(err)
	}
	ids := make([]int, 0, len(nonPayers))
	for _, p := range nonPayers {
		if p.Member != nil {
			ids = append(ids, p.Member.NetConfigID)
		}
	}
	err = s.netRepo.Deactivate(ctx, ids)
	if err != nil {
		return err
	}
	return s.UpdateDhcpFile(ctx)
}

func mapValidationError(err error) error {
	var fieldErrors validator.ValidationErrors
	if errors.As(err, &fieldErrors) {
		message := ""
		for _, fieldError := range fieldErrors {
			message += fmt.Sprintf("%s:%s-%s; ", fieldError.Field(), fieldError.Tag(), fieldError.Param())
		}
		return model.Error(http.StatusBadRequest, err.Error(), message)
	}
	return err
}
