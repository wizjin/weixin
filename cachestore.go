package weixin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type CacheStore interface {
	Set(k string, v AccessToken) error
	Get(k string) (AccessToken, error)
}
type MemoryCacheStore struct {
	mp map[string]AccessToken
	mu sync.Mutex
}

func NewMemoryCacheStore() *MemoryCacheStore {
	return &MemoryCacheStore{
		mp: make(map[string]AccessToken),
	}
}
func (s *MemoryCacheStore) Set(k string, v AccessToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mp[k] = v
	return nil
}
func (s *MemoryCacheStore) Get(k string) (AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.mp[k]
	if !ok {
		return AccessToken{}, nil
	}
	return v, nil
}

type RedisCacheStore struct {
	client *redisClient
}
type RedisOptions struct {
	Addrs     []string
	Password  string
	IsCluster bool
	DBNum     int
}
type redisClient struct {
	isCluster     bool
	client        *redis.Client
	clusterClient *redis.ClusterClient
}

func (c *redisClient) get(k string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	var res *redis.StringCmd
	if c.isCluster {
		res = c.clusterClient.Get(ctx, k)
	} else {
		res = c.client.Get(ctx, k)
	}
	if res.Err() != nil {
		return "", res.Err()
	}
	return res.Val(), nil
}
func (c *redisClient) set(k string, v interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	var res *redis.StatusCmd
	if c.isCluster {
		res = c.clusterClient.Set(ctx, k, v, 0)
	} else {
		res = c.client.Set(ctx, k, v, 0)
	}
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func NewRedisCacheStore(opt *RedisOptions) *RedisCacheStore {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	rc := &redisClient{}
	if opt.IsCluster {
		rc.isCluster = opt.IsCluster
		rc.clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    opt.Addrs,
			Password: opt.Password,
		})
		if _, err := rc.clusterClient.Ping(ctx).Result(); err != nil {
			panic(fmt.Sprintf("redis cluster ping failed: %s", err))
		}
	} else {
		rc.isCluster = opt.IsCluster
		rc.client = redis.NewClient(&redis.Options{
			Addr:     opt.Addrs[0],
			Password: opt.Password,
			DB:       opt.DBNum,
		})
		if _, err := rc.client.Ping(ctx).Result(); err != nil {
			panic(fmt.Sprintf("redis client ping failed: %s", err))
		}
	}
	return &RedisCacheStore{
		client: rc,
	}
}

func (s *RedisCacheStore) Set(k string, v AccessToken) error {
	val, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err = s.client.set(k, val); err != nil {
		return err
	}
	return nil
}
func (s *RedisCacheStore) Get(k string) (AccessToken, error) {
	res, err := s.client.get(k)
	if err != nil {
		return AccessToken{}, nil
	}
	var accToken AccessToken
	if err = json.Unmarshal([]byte(res), &accToken); err != nil {
		return AccessToken{}, nil
	}
	return accToken, nil
}
