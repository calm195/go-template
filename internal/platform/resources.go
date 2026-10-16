package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	rocketmq "github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/calm195/go-template/internal/config"
)

// Resources 管理可选中间件客户端及其生命周期。
type Resources struct {
	DB       *gorm.DB
	Redis    *redis.Client
	Producer rocketmq.Producer
}

// Open 初始化已启用的客户端，失败时释放已创建的资源。
func Open(ctx context.Context, c config.Config) (_ *Resources, err error) {
	r := &Resources{}
	defer func() {
		if err != nil {
			_ = r.Close()
		}
	}()
	if c.Database.Driver != "" {
		var dialector gorm.Dialector
		if c.Database.Driver == "mysql" {
			dialector = mysql.Open(c.Database.DSN)
		} else {
			dialector = postgres.Open(c.Database.DSN)
		}
		r.DB, err = gorm.Open(dialector, &gorm.Config{
			Logger:               logger.Default.LogMode(logger.Silent),
			DisableAutomaticPing: true,
		})
		if err != nil {
			return nil, fmt.Errorf("open database: %w", err)
		}
		db, e := r.DB.DB()
		if e != nil {
			return nil, e
		}
		db.SetMaxOpenConns(c.Database.MaxOpen)
		db.SetMaxIdleConns(c.Database.MaxIdle)
		db.SetConnMaxLifetime(c.Database.MaxLifetime)
		if err = db.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("ping database: %w", err)
		}
	}
	if c.Redis.Enabled {
		r.Redis = redis.NewClient(&redis.Options{
			Addr:         c.Redis.Address,
			Password:     c.Redis.Password,
			DB:           c.Redis.DB,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})
		if err = r.Redis.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("ping redis: %w", err)
		}
	}
	if c.RocketMQ.Enabled {
		r.Producer, err = rocketmq.NewProducer(
			producer.WithNameServer(primitive.NamesrvAddr(c.RocketMQ.NameServers)),
			producer.WithGroupName(c.RocketMQ.Group),
			producer.WithRetry(2),
		)
		if err != nil {
			return nil, fmt.Errorf("create producer: %w", err)
		}
		if err = r.Producer.Start(); err != nil {
			return nil, fmt.Errorf("start producer: %w", err)
		}
	}
	return r, nil
}

// Ready 检查已启用的数据库和 Redis 连接是否可用。
func (r *Resources) Ready(ctx context.Context) error {
	if r.DB != nil {
		db, err := r.DB.DB()
		if err != nil {
			return err
		}
		if err = db.PingContext(ctx); err != nil {
			return err
		}
	}
	if r.Redis != nil {
		return r.Redis.Ping(ctx).Err()
	}
	return nil
}

// Close 按初始化的逆序释放客户端。
func (r *Resources) Close() error {
	var errs []error
	if r.Producer != nil {
		errs = append(errs, r.Producer.Shutdown())
	}
	if r.Redis != nil {
		errs = append(errs, r.Redis.Close())
	}
	if r.DB != nil {
		db, err := r.DB.DB()
		if err != nil {
			errs = append(errs, err)
		} else {
			errs = append(errs, db.Close())
		}
	}
	return errors.Join(errs...)
}
