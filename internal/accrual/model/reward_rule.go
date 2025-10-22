package model

import "errors"

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
	Match      string     `json:"match"`
	Reward     float64    `json:"reward"`
	RewardType RewardType `json:"reward_type"`
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
