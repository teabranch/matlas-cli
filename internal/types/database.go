package types

import (
	"context"
	"time"
)

// DatabaseInfo represents information about a MongoDB database
type DatabaseInfo struct {
	Name        string           `json:"name"`
	SizeOnDisk  int64            `json:"sizeOnDisk"`
	Empty       bool             `json:"empty"`
	Collections []CollectionInfo `json:"collections,omitempty"`
}

// CollectionInfo represents information about a MongoDB collection
type CollectionInfo struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options,omitempty"`
	Info    CollectionStats        `json:"info,omitempty"`
}

// CollectionStats represents statistics about a collection
type CollectionStats struct {
	Count      int64            `json:"count"`
	Size       int64            `json:"size"`
	AvgObjSize int64            `json:"avgObjSize"`
	IndexSizes map[string]int64 `json:"indexSizes,omitempty"`
}

// IndexInfo represents information about a MongoDB index
type IndexInfo struct {
	Name       string                 `json:"name"`
	Keys       map[string]interface{} `json:"key"`
	Unique     bool                   `json:"unique,omitempty"`
	Sparse     bool                   `json:"sparse,omitempty"`
	Background bool                   `json:"background,omitempty"`
	Version    int                    `json:"v,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// ConnectionInfo represents connection details for a MongoDB instance
type ConnectionInfo struct {
	ConnectionString string            `json:"connectionString"`
	Database         string            `json:"database"`
	Options          map[string]string `json:"options,omitempty"`
	TempUser         *TempUserInfo     `json:"tempUser,omitempty"`
}

// TempUserInfo represents information about a temporary database user
type TempUserInfo struct {
	Username    string                      `json:"username"`
	ExpiresAt   time.Time                   `json:"expiresAt"`
	CleanupFunc func(context.Context) error `json:"-"` // Not serialized
}
