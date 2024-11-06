package cronjob

import (
	"context"
	"testing"

	mockLifecycle "github.com/ipfans/components/v2/mocks/lifecycle"
	mockRedis "github.com/ipfans/components/v2/mocks/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDistributedElector_Leader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name     string
		setup    func(mock *mockRedis.MockCmdable)
		wantErr  bool
		isLeader bool
	}{
		{
			name: "成功获取leader",
			setup: func(mock *mockRedis.MockCmdable) {
				// Mock SetNX for initial leader election
				setNXCmd := redis.NewBoolCmd(context.Background())
				setNXCmd.SetVal(true)
				mock.EXPECT().SetNX(gomock.Any(), LeaderKey, gomock.Any(), gomock.Any()).Return(setNXCmd).Times(1)

				// Mock Exists check in IsLeader
				existsCmd := redis.NewIntCmd(context.Background())
				existsCmd.SetVal(1) // Key exists
				mock.EXPECT().Exists(gomock.Any(), LeaderKey).Return(existsCmd).Times(1)

				// Mock Get check in IsLeader
				getCmd := redis.NewStringCmd(context.Background())
				getCmd.SetVal("test_id") // Should match currentID
				mock.EXPECT().Get(gomock.Any(), LeaderKey).Return(getCmd).Times(1)
			},
			wantErr:  false,
			isLeader: true,
		},
		{
			name: "获取leader失败",
			setup: func(mock *mockRedis.MockCmdable) {
				// Mock SetNX failure
				setNXCmd := redis.NewBoolCmd(context.Background())
				setNXCmd.SetErr(redis.ErrClosed)
				mock.EXPECT().SetNX(gomock.Any(), LeaderKey, gomock.Any(), gomock.Any()).Return(setNXCmd).Times(3)
			},
			wantErr:  true,
			isLeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := mockRedis.NewMockCmdable(ctrl)
			if tt.setup != nil {
				tt.setup(mockRedis)
			}

			elector := &DistributedElector{
				cmder:      mockRedis,
				currentID:  "test_id", // Set a fixed ID for testing
				defaultTTL: DefaultTTL,
			}

			// First try to become leader
			err := elector.lockLeader()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Then check if we are leader
			err = elector.IsLeader(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.isLeader, elector.isLeader)
		})
	}
}

func TestDistributedElector_Lock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name    string
		setup   func(mock *mockRedis.MockCmdable)
		key     string
		wantErr bool
	}{
		{
			name: "成功加锁",
			setup: func(mock *mockRedis.MockCmdable) {
				cmd := redis.NewBoolCmd(context.Background())
				cmd.SetVal(true) // SetNX 成功返回 true
				mock.EXPECT().SetNX(gomock.Any(), "test_lock", gomock.Any(), gomock.Any()).Return(cmd).Times(1)
			},
			key:     "test_lock",
			wantErr: false,
		},
		{
			name: "加锁失败",
			setup: func(mock *mockRedis.MockCmdable) {
				cmd := redis.NewBoolCmd(context.Background())
				cmd.SetErr(redis.ErrClosed)
				mock.EXPECT().SetNX(gomock.Any(), "test_lock", gomock.Any(), gomock.Any()).Return(cmd).Times(3)
			},
			key:     "test_lock",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := mockRedis.NewMockCmdable(ctrl)
			if tt.setup != nil {
				tt.setup(mockRedis)
			}

			elector := &DistributedElector{
				cmder:      mockRedis,
				currentID:  "test_id",
				defaultTTL: DefaultTTL,
			}

			lock, err := elector.Lock(context.Background(), tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, lock)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, lock)
			}
		})
	}
}

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name    string
		setup   func(mockRedis *mockRedis.MockCmdable, mockLC *mockLifecycle.MockLifecycle)
		wantErr bool
	}{
		{
			name: "成功创建scheduler",
			setup: func(mockRedis *mockRedis.MockCmdable, mockLC *mockLifecycle.MockLifecycle) {
				mockLC.EXPECT().Append(gomock.Any()).Times(1)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := mockRedis.NewMockCmdable(ctrl)
			mockLC := mockLifecycle.NewMockLifecycle(ctrl)

			if tt.setup != nil {
				tt.setup(mockRedis, mockLC)
			}

			scheduler, err := New(mockLC, mockRedis)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, scheduler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scheduler)
			}
		})
	}
}
