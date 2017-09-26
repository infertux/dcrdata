package dcrpg

import (
	"fmt"

	"github.com/dcrdata/dcrdata/db/dbtypes"
	"github.com/dcrdata/dcrdata/db/dcrpg/internal"
)

const (
	InsertBlockRow = `INSERT INTO blocks (
		hash, height, size, version, merkle_root, stake_root,
		numtx, num_rtx, tx, txDbIDs, num_stx, stx, stxDbIDs,
		time, nonce, vote_bits, final_state, voters,
		fresh_stake, revocations, pool_size, bits, sbits, 
		difficulty, extra_data, stake_version, previous_hash
	) VALUES ($1, $2, $3, $4, $5, $6, 
		$7, $8, %s, %s, $9, %s, %s,
		$10, $11, $12, $13, $14, 
		$15, $16, $17, $18, $19, 
		$20, $21, $22, $23) RETURNING id;`

	InsertTxRow = `INSERT INTO transactions (
		block_hash, block_index, tx_hash, version,
		lock_time, expiry, num_vin, vins,
		num_vout, vout_db_ids) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, %s) RETURNING id;`

	InsertVoutRow = `INSERT INTO vouts (outpoint, value, ind, version,
		pkscript, script_req_sigs, script_type, script_addresses)
		VALUES ($1, $2, $3, $4, $5, $6, $7, %s) RETURNING id;`
)

func makeBlockInsertStatement(txDbIDs, stxDbIDs []uint64, rtxs, stxs []string) string {
	txDbIDsARRAY := internal.MakeARRAYOfBIGINTs(txDbIDs)
	stxDbIDsARRAY := internal.MakeARRAYOfBIGINTs(stxDbIDs)
	rtxTEXTARRAY := internal.MakeARRAYOfTEXT(rtxs)
	stxTEXTARRAY := internal.MakeARRAYOfTEXT(stxs)
	return fmt.Sprintf(InsertBlockRow, rtxTEXTARRAY, txDbIDsARRAY,
		stxTEXTARRAY, stxDbIDsARRAY)
}

func MakeBlockInsertStatement(block *dbtypes.Block) string {
	return makeBlockInsertStatement(block.TxDbIDs, block.STxDbIDs,
		block.Tx, block.STx)
}

func makeVoutInsertStatement(scriptAddresses []string) string {
	addrs := internal.MakeARRAYOfTEXT(scriptAddresses)
	return fmt.Sprintf(InsertVoutRow, addrs)
}

func MakeVoutInsertStatement(vout *dbtypes.Vout) string {
	return makeVoutInsertStatement(vout.ScriptPubKeyData.Addresses)
}

func makeTxInsertStatement(voutDbIDs []uint64) string {
	dbIDsBIGINT := internal.MakeARRAYOfBIGINTs(voutDbIDs)
	return fmt.Sprintf(InsertTxRow, dbIDsBIGINT)
}

func MakeTxInsertStatement(tx *dbtypes.Tx) string {
	return makeTxInsertStatement(tx.VoutDbIds)
}
