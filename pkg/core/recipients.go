package core

import (
	"context"
	"net/http"
	"time"

	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RecipientsService struct {
	db *gorm.DB
}

func newRecipientsService(db *gorm.DB) *RecipientsService {
	return &RecipientsService{
		db: db,
	}
}

func (s *RecipientsService) Get(ctx context.Context, id string) (*Recipient, error) {
	recipient := new(Recipient)
	if err := s.db.Preload("Organization").Preload("BankAddress").Where("id = ?", id).First(recipient).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, cerr.NewHttpError(http.StatusNotFound, "recipient not found", err)
		}
		return nil, err
	}
	return recipient, nil
}

func (s *RecipientsService) List(ctx context.Context, req ListRecipientsRequest) (*ListRecipientsResponse, error) {
	recipients := make([]Recipient, 0)
	query := s.db
	if req.Ein != "" {
		query = query.Joins("Organization", query.Where("ein = ?", req.Ein)) //.Where("Organization.ein = ?", req.Ein)
	} else {
		query = query.Preload("Organization")
	}
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Cursor != "" {
		query = query.Where("id > ?", req.Cursor)
	}
	if err := query.Find(&recipients).Error; err != nil {
		return nil, err
	}

	nextCursor := ""
	if len(recipients) > 0 {
		nextCursor = recipients[len(recipients)-1].Id.String()
	}

	return &ListRecipientsResponse{
		Recipients: recipients,
		NextCursor: nextCursor,
	}, nil
}

type ListRecipientsRequest struct {
	Ein    string
	Limit  int
	Cursor string
}

type ListRecipientsResponse struct {
	Recipients []Recipient
	NextCursor string
}

// Recipient - A recipient is a verified nonprofit organization account that can receive payments.
type Recipient struct {
	Id               uuid.UUID `gorm:"primaryKey;uuid;not null"`
	Name             string    `gorm:"column:name;size=255"`
	Primary          bool
	OrganizationId   string        `gorm:"column:organization_id;size=255"`
	Organization     *Organization `gorm:"foreignKey:OrganizationId"`
	MailingAddressId uint64        `gorm:"column:mailing_address_id"`
	MailingAddress   *Address      `gorm:"foreignKey:MailingAddressId"`
	BankAddressId    uint64        `gorm:"column:bank_address_id"`
	BankAddress      *BankAddress  `gorm:"foreignKey:BankAddressId"`
	CreatedAt        time.Time     `gorm:"column:created_at"`
}

type Organization struct {
	Id            string   `gorm:"primaryKey;size:255"`
	LegalName     string   `gorm:"column:legal_name;size:255"`
	PreferredName string   `gorm:"column:preferred_name;size:255"`
	Ein           string   `gorm:"column:ein;size:9"`
	AddressId     uint64   `gorm:"column:address_id"`
	Address       *Address `gorm:"foreignKey:AddressId"`
}

type Address struct {
	Id         uint64 `gorm:"primaryKey"`
	Line1      string `gorm:"column:line1;size:255"`
	Line2      string `gorm:"column:line2;size:255"`
	City       string `gorm:"column:city;size:255"`
	State      string `gorm:"column:state;size:255"`
	PostalCode string `gorm:"column:postal_code;size:255"`
	Status     string `gorm:"column:status;size:255"`
	UpdatedAt  string `gorm:"column:updated_at;size:255"`
}

type BankAddress struct {
	Id            uint64 `gorm:"primaryKey"`
	AccountNumber string `gorm:"column:account_number;size:255"`
	RoutingNumber string `gorm:"column:routing_number;size:255"`
	Status        string `gorm:"column:status;size:255"`
	UpdatedAt     string `gorm:"column:updated_at;size:255"`
}
