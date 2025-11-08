package model

import (
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrRewardTypeInvalid = errors.New("invalid reward type")

type RewardType string

const (
	Percent RewardType = "%"
	Points  RewardType = "pt"
)

func (r RewardType) IsValid() bool {
	switch r {
	case Percent, Points:
		return true
	default:
		return false
	}
}

type RewardRule struct {
	Match      string             `json:"match" db:"match_pattern"`
	Reward     float64            `json:"reward" db:"reward"`
	RewardType RewardType         `json:"reward_type" db:"reward_type"`
	CreatedAt  pgtype.Timestamptz `json:"created_at" db:"created_at"`
}

func NewRewardRule(match string, reward float64, rewardType RewardType) (*RewardRule, error) {
	if !rewardType.IsValid() {
		return nil, ErrRewardTypeInvalid
	}
	return &RewardRule{
		Match:      match,
		Reward:     reward,
		RewardType: rewardType,
	}, nil
}
