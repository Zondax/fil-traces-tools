package types

type MultisigState struct {
	Height         int64    `json:"Height"`
	Signers        []string `json:"Signers"`
	LockedBalance  string   `json:"LockedBalance"`
	UnlockDuration int64    `json:"UnlockDuration"`
}
type Constructor struct {
	Signers        []string `json:"Signers"`
	Threshold      uint64   `json:"NumApprovalsThreshold"`
	LockedBalance  string   `json:"LockedBalance"`
	UnlockDuration int64    `json:"UnlockDuration"`
}

type AddSigner struct {
	Signer string `json:"Signer"`
}

type SwapSigner struct {
	From string `json:"From"`
	To   string `json:"To"`
}

type RemoveSigner struct {
	Signer string `json:"Signer"`
}

type LockBalance struct {
	Amount         string `json:"Amount"`
	StartEpoch     uint64 `json:"StartEpoch"`
	UnlockDuration int64  `json:"UnlockDuration"`
}

type SetThreshold struct {
	Threshold uint64 `json:"Threshold"`
}

type SetUnlockDuration struct {
	UnlockDuration int64 `json:"UnlockDuration"`
}
