package varys

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

// Queue interface
type Queue interface {
	Enqueue(urls ...string) error
	Dequeue() (string, error)

	Repaire() error

	DoneURL(url string) error
	RetryURL(url string) error

	FailedURLs() []string

	Cleanup()
}

// RedisQueue is an redis-based implementation of Queue interface.
type RedisQueue struct {
	pool *redis.Pool

	QueueReady   string
	QueuePending string
	QueueDone    string
	QueueFailed  string
}

// NewRedisQueue creates a new RedisQueue instance.
func NewRedisQueue(url, password, prefix string) *RedisQueue {
	return &RedisQueue{
		pool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				conn, err := redis.Dial("tcp", url)
				if err != nil {
					return nil, err
				}
				if len(password) == 0 {
					return conn, err
				}
				if _, err = conn.Do("AUTH", password); err != nil {
					conn.Close()
					return nil, err
				}
				return conn, nil
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
			Wait: true,
		},
		QueueReady:   prefix + "queue-ready",
		QueuePending: prefix + "queue-pending",
		QueueDone:    prefix + "queue-done",
		QueueFailed:  prefix + "queue-failed",
	}
}

// Enqueue adds urls into ready queue.
func (q *RedisQueue) Enqueue(urls ...string) error {
	conn := q.pool.Get()
	defer conn.Close()

	for _, url := range urls {
		if dup, err := redis.Bool(conn.Do("SISMEMBER", q.QueueFailed, url)); err == nil && dup {
			continue
		}
		if dup, err := redis.Bool(conn.Do("SISMEMBER", q.QueueDone, url)); err == nil && dup {
			continue
		}
		if dup, err := redis.Bool(conn.Do("SISMEMBER", q.QueuePending, url)); err == nil && dup {
			continue
		}
		_, err := conn.Do("SADD", q.QueueReady, url)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *RedisQueue) Dequeue() (url string, err error) {
	conn := q.pool.Get()
	defer conn.Close()

	url, err = redis.String(conn.Do("SPOP", q.QueueReady))
	if err != nil {
		return "", nil
	}
	_, err = conn.Do("SADD", q.QueuePending, url)

	return
}

func (q *RedisQueue) Repaire() error {
	conn := q.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SUNIONSTORE", q.QueueReady, q.QueueReady, q.QueuePending)
	return err
}

func (q *RedisQueue) DoneURL(url string) error {
	conn := q.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", q.QueuePending, q.QueueDone, url)
	return err
}

func (q *RedisQueue) RetryURL(url string) error {
	conn := q.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", q.QueuePending, q.QueueFailed, url)
	return err
}

func (q *RedisQueue) FailedURLs() []string {
	conn := q.pool.Get()
	defer conn.Close()
	urls, err := redis.Strings(conn.Do("SMEMBERS", q.QueueFailed))
	if err != nil {
		return nil
	}
	return urls
}

func (q *RedisQueue) Cleanup() {

}
