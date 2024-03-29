package conf

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
)

var (
	global *Config
	DB     *sql.DB
)

type Config struct {
	Mysql *Mysql `toml:"mysql"`
}

func NewConfig() *Config {
	return &Config{
		Mysql: newDefaultMysql(),
	}
}

func (c *Config) InitGlobal() error {
	global = c
	return nil
}

func LoadConfigFromToml(filePath string) error {
	var err error
	cfg := NewConfig()
	if _, err = toml.DecodeFile(filePath, cfg); err != nil {
		return err
	}
	global = cfg
	return cfg.InitGlobal()
}

func C() *Config {
	if global == nil {
		err := LoadConfigFromToml("../etc/app.toml")
		if err != nil {
			fmt.Println(err)
		}
		//LoadConfigFromToml("../../../etc/app.toml")
	}
	return global
}

type Mysql struct {
	Host        string `toml:"host"`
	Port        string `toml:"port"`
	Username    string `toml:"username"`
	Password    string `toml:"password"`
	MaxOpenConn int    `toml:"max_open_conn"`
	MaxIdleConn int    `toml:"max_idle_conn"`
	MaxLifeTime int    `toml:"max_life_time"`
	maxIdleTime int    `toml:"max_idle_time"`
	lock        sync.Mutex

	Database string `toml:"database"`
}

func newDefaultMysql() *Mysql {
	return &Mysql{
		Host:        "127.0.0.1",
		Port:        "3306",
		MaxOpenConn: 200,
		MaxIdleConn: 100,

		Database: "min_data",
	}
}

func (m *Mysql) GetDB() *sql.DB {
	m.lock.Lock()
	defer m.lock.Unlock()
	if DB == nil {
		conn, err := m.GetDBConn(m.Database)
		if err != nil {
			panic(err)
		}
		DB = conn
	}
	return DB
}

func (m *Mysql) GetDBConn(database string) (*sql.DB, error) {
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&multiStatements=true", m.Username, m.Password, m.Host, m.Port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to mysql <%s> error,%s", dsn, err.Error())
	}
	db.SetMaxOpenConns(m.MaxOpenConn)
	db.SetMaxIdleConns(m.MaxIdleConn)
	db.SetConnMaxLifetime(time.Second * time.Duration(m.MaxLifeTime))
	db.SetConnMaxIdleTime(time.Second * time.Duration(m.maxIdleTime))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql <%s> error,%s", dsn, err.Error())
	}
	return db, nil
}
