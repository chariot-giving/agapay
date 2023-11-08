package core

import (
	"github.com/chariot-giving/agapay/pkg/bank"
	"gorm.io/gorm"
)

type AgapayServer struct {
	db   *gorm.DB
	bank bank.Bank
}

func NewAgapayServer(db *gorm.DB, bank bank.Bank) *AgapayServer {
	return &AgapayServer{
		db:   db,
		bank: bank,
	}
}
