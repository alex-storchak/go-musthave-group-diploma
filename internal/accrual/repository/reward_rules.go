package repository

import (
	"context"
	"fmt"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewPgRewardRules(pool *pgxpool.Pool, l *zap.Logger) *PgRewardRules {
	return &PgRewardRules{
		dbPool: pool,
		logger: l,
	}
}

type PgRewardRules struct {
	dbPool *pgxpool.Pool
	logger *zap.Logger
}

func (p *PgRewardRules) Add(ctx context.Context, rule *model.RewardRule) error {
	q := `
		INSERT INTO reward_rules (match_pattern, reward, reward_type_id) 
		VALUES ($1, $2, (SELECT id from reward_types rt WHERE rt.code = $3))
	`

	_, err := p.dbPool.Exec(ctx, q, rule.Match, rule.Reward, rule.RewardType)
	if err != nil {
		return fmt.Errorf("insert reward rule(%s, %f, %s): %w", rule.Match, rule.Reward, rule.RewardType, err)
	}
	return nil
}

func (p *PgRewardRules) Has(ctx context.Context, match string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM reward_rules rr WHERE rr.match_pattern = $1)`
	var has bool
	row := p.dbPool.QueryRow(ctx, q, match)
	err := row.Scan(&has)
	if err != nil {
		return false, fmt.Errorf("scan query result row: %w", err)
	}
	return has, nil
}

func (p *PgRewardRules) All(ctx context.Context) ([]model.RewardRule, error) {
	q := `
		SELECT rr.match_pattern, rr.reward, rt.code as reward_type, rr.created_at 
		FROM reward_rules rr
		JOIN reward_types rt ON rr.reward_type_id = rt.id
	`
	rows, err := p.dbPool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query all reward rules: %w", err)
	}
	defer rows.Close()

	estimateRulesCount := 1024
	rules := make([]model.RewardRule, 0, estimateRulesCount)
	for rows.Next() {
		var rule model.RewardRule
		err = rows.Scan(&rule.Match, &rule.Reward, &rule.RewardType, &rule.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan reward rule: %w", err)
		}
		rules = append(rules, rule)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("scan all reward rules from db: %w", err)
	}

	return rules, nil
}

func (p *PgRewardRules) Close() {
	p.dbPool.Close()
}
