package atomic

type RecoveryPoint string

const (
	RecoveryPointStarted            RecoveryPoint = "started"
	RecoveryPointFinished           RecoveryPoint = "finished"
	RecoveryPointAccountCreated     RecoveryPoint = "account_created"
	RecoveryPointBankAccountCreated RecoveryPoint = "bank_account_created"
)
