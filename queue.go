package varys

import "github.com/garyburd/redigo/redis"

const (
	queueReady   = "queue-ready"
	queuePending = "queue-pending"
	queueDone    = "queue-done"
	queueFailed  = "queue-failed"
)

type Queue interface {
	Enqueue(urls ...string) error
	Dequeue() (string, error)

	Repaire() error

	DoneURL(url string) error
	RetryURL(url string) error

	Cleanup()
}

type RedisQueue struct {
	pool *redis.Pool
}

func NewRedisQueue() *RedisQueue {
	return &RedisQueue{
		pool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", "127.0.0.1:6379")
			},
			Wait: true,
		},
	}
}

func (q *RedisQueue) Enqueue(urls ...string) error {
	conn := q.pool.Get()
	defer conn.Close()

	for _, url := range urls {
		if dup, err := redis.Bool(conn.Do("SISMEMBER", queueFailed, url)); err == nil && dup {
			continue
		}
		if dup, err := redis.Bool(conn.Do("SISMEMBER", queueDone, url)); err == nil && dup {
			continue
		}
		if dup, err := redis.Bool(conn.Do("SISMEMBER", queuePending, url)); err == nil && dup {
			continue
		}
		_, err := conn.Do("SADD", queueReady, url)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *RedisQueue) Dequeue() (url string, err error) {
	conn := q.pool.Get()
	defer conn.Close()

	url, err = redis.String(conn.Do("SPOP", queueReady))
	if err != nil {
		return "", nil
	}
	_, err = conn.Do("SADD", queuePending, url)

	return
}

func (q *RedisQueue) Repaire() error {
	conn := q.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SUNIONSTORE", queueReady, queueReady, queuePending)
	return err
}

func (q *RedisQueue) DoneURL(url string) error {
	conn := q.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", queuePending, queueDone, url)
	return err
}

func (q *RedisQueue) RetryURL(url string) error {
	conn := q.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", queuePending, queueFailed, url)
	return err
}

func (q *RedisQueue) Cleanup() {

}
