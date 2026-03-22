package client

import (
	"context"
	"encoding/json"
)

// DoctorClient provides doctor/diagnostic operations via the daemon.
type DoctorClient struct {
	client *Client
}

// DoctorReport represents the full doctor report returned by the daemon.
type DoctorReport struct {
	Categories []DoctorCategory `json:"categories"`
}

// DoctorCategory represents a single category within a doctor report.
type DoctorCategory struct {
	Name  string       `json:"name"`
	Items []DoctorItem `json:"items"`
}

// DoctorItem represents a single check item within a doctor category.
type DoctorItem struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// RunAll runs all doctor checks and returns the full report.
func (d *DoctorClient) RunAll(ctx context.Context) (*DoctorReport, error) {
	var report DoctorReport
	if err := d.client.get(ctx, "/api/doctor", &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// ByCategory runs doctor checks for a specific category.
func (d *DoctorClient) ByCategory(ctx context.Context, cat string) (*DoctorCategory, error) {
	var category DoctorCategory
	if err := d.client.get(ctx, "/api/doctor/"+cat, &category); err != nil {
		return nil, err
	}
	return &category, nil
}

// RunAllRaw runs all doctor checks and returns the raw JSON.
func (d *DoctorClient) RunAllRaw(ctx context.Context) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := d.client.get(ctx, "/api/doctor", &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
