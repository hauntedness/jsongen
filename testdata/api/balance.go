package api

type Details struct {
	AvailEq    string `json:"availEq"`
	BorrowFroz string `json:"borrowFroz"`
	CashBal    string `json:"cashBal"`
}

type First struct {
	AvailBal  string `json:"availBal"`
	Ccy       string `json:"ccy"`
	CrossLiab string `json:"crossLiab"`
}

type Balance struct {
	AdjEq   string    `json:"adjEq"`
	Details []Details `json:"details"`
	First   First     `json:"first"`
	IsoEq   any       `json:"isoEq"`
	Numbers []float64 `json:"numbers"`
	UTime   float64   `json:"uTime"`
	Upl     float64   `json:"upl"`
}
