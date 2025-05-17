package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/AliRizaAynaci/gorl/storage"
	goredis "github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *goredis.Client
	ctx    context.Context
}

func NewRedisStore(redisURL string) storage.Storage {
	opt, err := goredis.ParseURL(redisURL)
	if err != nil {
		panic(err)
	}
	client := goredis.NewClient(opt)
	return &RedisStore{
		client: client,
		ctx:    context.Background(),
	}
}

// ----- Simple counter ops -----

func (s *RedisStore) Incr(key string, ttl time.Duration) (float64, error) {
	val, err := s.client.Incr(s.ctx, key).Result()
	if err != nil {
		return 0, err
	}
	_, _ = s.client.Expire(s.ctx, key, ttl).Result()
	return float64(val), nil
}

func (s *RedisStore) Get(key string) (float64, error) {
	val, err := s.client.Get(s.ctx, key).Result()
	if err == goredis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	floatVal, _ := strconv.ParseFloat(val, 64)
	return floatVal, nil
}

func (s *RedisStore) Set(key string, val float64, ttl time.Duration) error {
	return s.client.Set(s.ctx, key, val, ttl).Err()
}

// ----- List ops (for sliding window) -----

func (s *RedisStore) AppendList(key string, value int64, ttl time.Duration) error {
	if err := s.client.RPush(s.ctx, key, value).Err(); err != nil {
		return err
	}
	_, _ = s.client.Expire(s.ctx, key, ttl).Result()
	return nil
}

func (s *RedisStore) GetList(key string) ([]int64, error) {
	vals, err := s.client.LRange(s.ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	result := make([]int64, 0, len(vals))
	for _, v := range vals {
		intVal, _ := strconv.ParseInt(v, 10, 64)
		result = append(result, intVal)
	}
	return result, nil
}

func (s *RedisStore) TrimList(key string, count int) error {
	// Keep only the last 'count' items (trim to size)
	length, err := s.client.LLen(s.ctx, key).Result()
	if err != nil {
		return err
	}
	start := length - int64(count)
	if start < 0 {
		start = 0
	}
	return s.client.LTrim(s.ctx, key, start, -1).Err()
}

// ----- Sorted set ops (for precise sliding window) -----

func (s *RedisStore) ZAdd(key string, score float64, member int64, ttl time.Duration) error {
	if err := s.client.ZAdd(s.ctx, key, goredis.Z{Score: score, Member: member}).Err(); err != nil {
		return err
	}
	_, _ = s.client.Expire(s.ctx, key, ttl).Result()
	return nil
}

func (s *RedisStore) ZRemRangeByScore(key string, min, max float64) error {
	return s.client.ZRemRangeByScore(s.ctx, key, fmt.Sprintf("%f", min), fmt.Sprintf("%f", max)).Err()
}

func (s *RedisStore) ZCard(key string) (int64, error) {
	return s.client.ZCard(s.ctx, key).Result()
}

func (s *RedisStore) ZRangeByScore(key string, min, max float64) ([]int64, error) {
	vals, err := s.client.ZRangeByScore(s.ctx, key, &goredis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}).Result()
	if err != nil {
		return nil, err
	}
	result := make([]int64, 0, len(vals))
	for _, v := range vals {
		intVal, _ := strconv.ParseInt(v, 10, 64)
		result = append(result, intVal)
	}
	return result, nil
}

// ----- Hash ops (for complex state) -----

func (s *RedisStore) HMSet(key string, fields map[string]float64, ttl time.Duration) error {
	stringFields := make(map[string]interface{})
	for k, v := range fields {
		stringFields[k] = v
	}
	if err := s.client.HSet(s.ctx, key, stringFields).Err(); err != nil {
		return err
	}
	_, _ = s.client.Expire(s.ctx, key, ttl).Result()
	return nil
}

func (s *RedisStore) HMGet(key string, fields ...string) (map[string]float64, error) {
	vals, err := s.client.HMGet(s.ctx, key, fields...).Result()
	if err != nil {
		return nil, err
	}
	res := make(map[string]float64)
	for i, v := range vals {
		if v == nil {
			res[fields[i]] = 0
		} else {
			floatVal, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
			res[fields[i]] = floatVal
		}
	}
	return res, nil
}

func (s *RedisStore) Client() *goredis.Client {
	return s.client
}
