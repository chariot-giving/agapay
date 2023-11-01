package network

var PayeeDB PayeeDatabase

type PayeeDatabase interface {
	// GetPayeeElectronicAccount returns the electronic account for the given payeeID.
	GetPayeeElectronicAccount(payeeID string) (PayeeElectronicAccount, error)
	// CreatePayeeElectronicAccount creates a new electronic account for the given payeeID.
	CreatePayeeElectronicAccount(payeeID string, account PayeeElectronicAccount) error
}

func NewPayeeDatabase() PayeeDatabase {
	return newInMemoryPayeeDatabase()
}

type PayeeElectronicAccount struct {
	Name          string
	AccountNumber string
	RoutingNumber string
}

type ErrPayeeNotFound struct{}

func (e ErrPayeeNotFound) Error() string {
	return "payee not found"
}
