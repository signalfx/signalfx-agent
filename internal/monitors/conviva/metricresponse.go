package conviva

type metricResponse struct {
	Type              string               `json:"type"`
	FilterIDValuesMap map[string][]float64 `json:"filters,omitempty"`
	Meta              *meta                `json:"meta"`
	Tables            map[string]table     `json:"tables,omitempty"`
	Timestamps        []float64            `json:"timestamps,omitempty"`
	Xvalues           []string             `json:"xvalues,omitempty"`
}

type table struct {
	Rows     [][]float64 `json:"rows,omitempty"`
	TotalRow []float64   `json:"total_row,omitempty"`
}

type meta struct {
	Status                float64   `json:"status,omitempty"`
	FiltersWarmup         []float64 `json:"filters_warmup,omitempty"`
	FiltersNotExist       []float64 `json:"filters_not_exist,omitempty"`
	FiltersIncompleteData []float64 `json:"filters_incomplete_data,omitempty"`
}
