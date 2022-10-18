// Copyright 2021 gotomicro
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build e2e

package rlock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type ClientE2ESuite struct {
	suite.Suite
	rdb redis.Cmdable
}

func (s *ClientE2ESuite) SetupSuite() {
	s.rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	// 确保测试的目标 Redis 已经启动成功了
	for s.rdb.Ping(context.Background()).Err() != nil {

	}
}

func TestClientE2E(t *testing.T) {
	suite.Run(t, &ClientE2ESuite{})
}

func (s *ClientE2ESuite) TestLock() {
	t := s.T()
	rdb := s.rdb
	testCases := []struct {
		name string

		key        string
		expiration time.Duration
		retry      RetryStrategy
		timeout    time.Duration

		client *Client

		wantLock *Lock
		wantErr  error

		before func()
		after  func()
	}{
		{
			// 一次加锁成功
			name:       "locked",
			key:        "locked-key",
			retry:      &FixIntervalRetry{Interval: time.Second, Max: 3},
			timeout:    time.Second,
			expiration: time.Minute,
			client:     NewClient(rdb),
			before:     func() {},
			after: func() {
				res, err := rdb.Del(context.Background(), "locked-key").Result()
				require.NoError(t, err)
				require.Equal(t, int64(1), res)
			},
		},
		{
			// 加锁不成功
			name:       "failed",
			key:        "failed-key",
			retry:      &FixIntervalRetry{Interval: time.Second, Max: 3},
			timeout:    time.Second,
			expiration: time.Minute,
			client:     NewClient(rdb),
			before: func() {
				res, err := rdb.Set(context.Background(), "failed-key", "123", time.Minute).Result()
				require.NoError(t, err)
				require.Equal(t, "OK", res)
			},
			after: func() {
				res, err := rdb.Get(context.Background(), "failed-key").Result()
				require.NoError(t, err)
				require.Equal(t, "123", res)
				delRes, err := rdb.Del(context.Background(), "failed-key").Result()
				require.NoError(t, err)
				require.Equal(t, int64(1), delRes)
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			// 已经加锁，再加锁就是刷新超时时间
			name:       "already-locked",
			key:        "already-key",
			retry:      &FixIntervalRetry{Interval: time.Second, Max: 3},
			timeout:    time.Second,
			expiration: time.Minute,
			client: func() *Client {
				client := NewClient(rdb)
				client.valuer = func() string {
					return "123"
				}
				return client
			}(),
			before: func() {
				res, err := rdb.Set(context.Background(), "already-key", "123", time.Minute).Result()
				require.NoError(t, err)
				require.Equal(t, "OK", res)
			},
			after: func() {
				res, err := rdb.Get(context.Background(), "already-key").Result()
				require.NoError(t, err)
				require.Equal(t, "123", res)
				delRes, err := rdb.Del(context.Background(), "already-key").Result()
				require.NoError(t, err)
				require.Equal(t, int64(1), delRes)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			l, err := tc.client.Lock(context.Background(), tc.key, tc.expiration, tc.retry, tc.timeout)
			assert.True(t, errors.Is(err, tc.wantErr))
			if err != nil {
				return
			}
			assert.Equal(t, tc.key, l.key)
			assert.Equal(t, tc.expiration, l.expiration)
			assert.NotEmpty(t, l.value)
			tc.after()
		})
	}
}

//func (s *ClientE2ESuite) TestTryLock() {
//	t := s.T()
//	rdb := s.rdb
//	client := NewClient(rdb)
//	testCases := []struct {
//		name string
//
//		key        string
//		expiration time.Duration
//
//		wantLock *Lock
//		wantErr  error
//
//		before func()
//		after  func()
//	}{
//		{
//			// 加锁成功
//			name:       "locked",
//			key:        "locked-key",
//			expiration: time.Minute,
//			before:     func() {},
//			after: func() {
//				res, err := rdb.Del(context.Background(), "locked-key").Result()
//				require.NoError(t, err)
//				require.Equal(t, int64(1), res)
//			},
//			wantLock: &Lock{
//				key:        "locked-key",
//				expiration: time.Minute,
//			},
//		},
//		{
//			// 模拟并发竞争失败
//			name:       "failed",
//			key:        "failed-key",
//			expiration: time.Minute,
//			before: func() {
//				// 假设已经有人设置了分布式锁
//				val, err := rdb.Set(context.Background(), "failed-key", "123", time.Minute).Result()
//				require.NoError(t, err)
//				require.Equal(t, "OK", val)
//			},
//			after: func() {
//				res, err := rdb.Del(context.Background(), "failed-key").Result()
//				require.NoError(t, err)
//				require.Equal(t, int64(1), res)
//			},
//			wantErr: ErrFailedToPreemptLock,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			tc.before()
//			l, err := client.TryLock(context.Background(), tc.key, tc.expiration)
//			assert.Equal(t, tc.wantErr, err)
//			if err != nil {
//				return
//			}
//			assert.Equal(t, tc.key, l.key)
//			assert.Equal(t, tc.expiration, l.expiration)
//			assert.NotEmpty(t, l.value)
//			tc.after()
//		})
//	}
//}

func (s *ClientE2ESuite) TestUnLock(t *testing.T) {
	rdb := s.rdb
	client := NewClient(rdb)
	testCases := []struct {
		name string

		lock *Lock

		before func()
		after  func()

		wantLock *Lock
		wantErr  error
	}{
		{
			name: "unlocked",
			lock: func() *Lock {
				l, err := client.TryLock(context.Background(), "unlocked-key", time.Minute)
				require.NoError(t, err)
				return l
			}(),
			before: func() {},
			after: func() {
				res, err := rdb.Exists(context.Background(), "unlocked-key").Result()
				require.NoError(t, err)
				require.Equal(t, 1, res)
			},
		},
		{
			name:    "lock not hold",
			lock:    newLock(rdb, "not-hold-key", "123", time.Minute),
			wantErr: ErrLockNotHold,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.lock.Unlock(context.Background())
			require.Equal(t, tc.wantErr, err)
		})
	}
}
