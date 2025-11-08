package service

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func rulesA() []model.RewardRule {
	return []model.RewardRule{
		{Match: "A", Reward: 1.0, RewardType: model.Percent},
	}
}

func rulesB() []model.RewardRule {
	return []model.RewardRule{
		{Match: "B", Reward: 2.0, RewardType: model.Points},
	}
}

func TestCacheRefresherFillsEmptyCache(t *testing.T) {
	logger := zap.NewNop()
	repo := mocks.NewMockRulesRepository(t)

	ttl := 50 * time.Millisecond
	p := NewCacheRulesProvider(repo, ttl, logger)

	repo.EXPECT().
		All(mock.Anything).
		Return(rulesA(), nil).
		Maybe()

	ctx, cancel := context.WithCancel(t.Context())
	p.Start(ctx)

	time.Sleep(100 * time.Millisecond)
	got, err := p.All(ctx)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "A", got[0].Match)

	cancel()
	p.Close()
}

func TestAllReturnsFromCacheWhenClean(t *testing.T) {
	logger := zap.NewNop()
	repo := mocks.NewMockRulesRepository(t)

	ttl := time.Minute
	p := NewCacheRulesProvider(repo, ttl, logger)

	repo.EXPECT().
		All(mock.Anything).
		Return(rulesA(), nil).
		Once()

	got1, err := p.All(t.Context())
	require.NoError(t, err)
	require.Len(t, got1, 1)
	require.Equal(t, "A", got1[0].Match)

	got2, err := p.All(context.Background())
	require.NoError(t, err)
	require.Equal(t, got1, got2)
}

func TestAllFetchesFromRepoWhenDirty(t *testing.T) {
	logger := zap.NewNop()
	repo := mocks.NewMockRulesRepository(t)

	ttl := time.Minute
	p := NewCacheRulesProvider(repo, ttl, logger)

	repo.EXPECT().
		All(mock.Anything).
		Return(rulesA(), nil).
		Once()
	got1, err := p.All(t.Context())
	require.NoError(t, err)
	require.Len(t, got1, 1)
	require.Equal(t, "A", got1[0].Match)

	p.MarkDirty()
	repo.EXPECT().
		All(mock.Anything).
		Return(rulesB(), nil).
		Once()

	got2, err := p.All(t.Context())
	require.NoError(t, err)
	require.Len(t, got2, 1)
	require.Equal(t, "B", got2[0].Match)

	got3, err := p.All(context.Background())
	require.NoError(t, err)
	require.Equal(t, got2, got3)
}

func TestCloseWaitsRefresher(t *testing.T) {
	logger := zap.NewNop()
	repo := mocks.NewMockRulesRepository(t)

	ttl := 100 * time.Millisecond
	p := NewCacheRulesProvider(repo, ttl, logger)

	repo.EXPECT().
		All(mock.Anything).
		Return(rulesA(), nil).
		Maybe()

	ctx, cancel := context.WithCancel(t.Context())
	p.Start(ctx)

	time.Sleep(time.Second)

	cancel()
	p.Close()

	require.True(t, true)
}
