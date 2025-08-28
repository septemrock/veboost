package psql

import "github.com/lib/pq"

type AirdropData struct {
	ID        int64  `gorm:"primary_key" json:"id"`
	CreatedAt int64  `gorm:"not null;default:0" json:"createdAt"`
	UpdatedAt int64  `gorm:"not null;default:0" json:"updatedAt"`
	Epoch     uint64 `gorm:"not null;default:0;index:idx_epoch_user" json:"epoch"`
	Address   string `gorm:"not null;index:idx_epoch_user" json:"address"`
	Amount    string `gorm:"type:numeric(78,0);not null" json:"amount"`
	// Proof must be of type pq.StringArray, not []string.
	// Although the underlying type is []string, pq.StringArray provides the necessary interface
	// for GORM to correctly translate the data into a SQL statement for PostgreSQL array types.
	Proof   pq.StringArray `gorm:"type:text[];default:'{}'" json:"proof"`
	Claimed bool           `gorm:"not null;default:false" json:"claimed"`
}

type MerkleTreeData struct {
	ID            int64  `gorm:"primary_key" json:"id"`
	Epoch         uint64 `gorm:"not null;default:0;index:idx_epoch,unique" json:"epoch"`
	MarshaledTree string `gorm:"type:text;not null" json:"marshaledTree"`
}
