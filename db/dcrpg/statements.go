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

func MakeBlockInsertStatement(block *dbtypes.Block, checked bool) string {
	return internal.MakeBlockInsertStatement(block.TxDbIDs, block.STxDbIDs,
		block.Tx, block.STx, checked)
}

func MakeVoutInsertStatement(vout *dbtypes.Vout, checked bool) string {
	return internal.MakeVoutInsertStatement(vout.ScriptPubKeyData.Addresses, checked)
}

func MakeTxInsertStatement(tx *dbtypes.Tx, checked bool) string {
	return internal.MakeTxInsertStatement(tx.VoutDbIds, tx.VinDbIds, checked)
}
