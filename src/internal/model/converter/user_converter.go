package converter

import (
	"notification-service/src/internal/entity"
	"notification-service/src/internal/model"
)

func UserToResponse(user *entity.User) *model.UserResponse {
	return &model.UserResponse{
		ID:           user.UserID,
		Name:         user.FullName,
		MobileNumber: user.MobileNumber,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}
