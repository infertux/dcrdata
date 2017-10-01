package dcrpg

import (
	"github.com/dcrdata/dcrdata/db/dbtypes"
	"github.com/dcrdata/dcrdata/db/dcrpg/internal"
)

const (
	insertBlockPrevNext = `INSERT INTO block_chain (
		block_db_id, prev_hash, this_hash, next_hash)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (this_hash) DO NOTHING;`

	updateBlockNext = `UPDATE block_chain set next_hash = $2 WHERE block_db_id = $1;`
)

func MakeBlockInsertStatement(block *dbtypes.Block) string {
	return internal.MakeBlockInsertStatement(block.TxDbIDs, block.STxDbIDs,
		block.Tx, block.STx)
}

func MakeVoutInsertStatement(vout *dbtypes.Vout) string {
	return internal.MakeVoutInsertStatement(vout.ScriptPubKeyData.Addresses)
}

func MakeTxInsertStatement(tx *dbtypes.Tx) string {
	return internal.MakeTxInsertStatement(tx.VoutDbIds)
}
