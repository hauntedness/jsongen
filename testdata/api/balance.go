package api

import (
	"encoding/json"
)

var _ = json.Number

type Details struct {
	AvailEq    string `json:"availEq"`
	BorrowFroz string `json:"borrowFroz"`
	CashBal    string `json:"cashBal"`
}

type First struct {
	AvailBal  string `json:"availBal"`
	Ccy       string `json:"ccy"`
	CrossLiab string `json:"cross_liab"`
}

type Balance struct {
	AdjEq   string        `json:"adjEq"`
	Details []Details     `json:"details"`
	First   First         `json:"first"`
	IsoEq   any           `json:"isoEq"`
	Numbers []json.Number `json:"numbers"`
	UTime   json.Number   `json:"uTime"`
	Upl     json.Number   `json:"upl"`
}
