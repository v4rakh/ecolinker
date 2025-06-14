package dto

type EcoFlowHistoryItem struct {
	IndexName  string   `json:"indexName"`
	IndexValue *float64 `json:"indexValue,omitempty"`
	Unit       string   `json:"unit"`
}

type EcoFlowHistoryData = []EcoFlowHistoryItem

type EcoFlowDeviceItem struct {
	SN     string `json:"sn"`
	Online int    `json:"online"`
}

type EcoFlowDeviceData = []EcoFlowDeviceItem

type CollectorEcoFlowHttpDeviceParameterPayload struct {
	Parameters []string `json:"parameters"`
}
