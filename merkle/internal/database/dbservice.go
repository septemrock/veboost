package database

import (
	"fmt"

	"github.com/Bedrock-Technology/VeMerkle/internal/config"
	"github.com/Bedrock-Technology/VeMerkle/internal/database/psql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbInstances = make(map[string]*gorm.DB)

func GetDBConnection(dbType string) (*gorm.DB, error) {
	cfg := config.GetConfig()
	var dsn string

	switch dbType {
	case "postgres":
		dsn = cfg.DBServers.VeDsnPsql
	case "mysql":
		dsn = cfg.DBServers.VeDsnMysql
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	if dsn == "" {
		return nil, fmt.Errorf("dsn not configured for %s", dbType)
	}

	if db, exists := dbInstances[dsn]; exists {
		return db, nil
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %s", err)
	}

	dbInstances[dsn] = db
	return db, nil
}

func InitPostgres() {
	dsn := config.GetConfig().DBServers.VeDsnPsql
	db, err := gorm.Open(postgres.Open(config.GetConfig().DBServers.VeDsnPsql), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect to database: %s", dsn))
	}
	db.AutoMigrate(psql.AirdropData{})
	db.AutoMigrate(psql.MerkleTreeData{})
	dbInstances["postgres"] = db
}
