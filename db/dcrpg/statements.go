package dcrpg

import (
	"github.com/dcrdata/dcrdata/db/dbtypes"
	"github.com/dcrdata/dcrdata/db/dcrpg/internal"
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
