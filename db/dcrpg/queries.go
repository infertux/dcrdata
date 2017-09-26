package dcrpg

import (
	"database/sql"
	"fmt"

	"github.com/dcrdata/dcrdata/db/dbtypes"
)

func InsertBlock(db *sql.DB, dbBlock *dbtypes.Block) (uint64, error) {
	insertStatement := MakeBlockInsertStatement(dbBlock)
	var id uint64
	err := db.QueryRow(insertStatement,
		dbBlock.Hash, dbBlock.Height, dbBlock.Size, dbBlock.Version,
		dbBlock.MerkleRoot, dbBlock.StakeRoot,
		dbBlock.NumTx, dbBlock.NumRegTx, dbBlock.NumStakeTx,
		dbBlock.Time, dbBlock.Nonce, dbBlock.VoteBits,
		dbBlock.FinalState, dbBlock.Voters, dbBlock.FreshStake,
		dbBlock.Revocations, dbBlock.PoolSize, dbBlock.Bits,
		dbBlock.SBits, dbBlock.Difficulty, dbBlock.ExtraData,
		dbBlock.StakeVersion, dbBlock.PreviousHash).Scan(&id)
	return id, err
}

func InsertBlockPrevNext(db *sql.DB, block_db_id uint64,
	hash, prev, next string) error {
	rows, err := db.Query(insertBlockPrevNext, block_db_id, prev, hash, next)
	_ = rows.Close()
	return err
}

func UpdateBlockNext(db *sql.DB, block_db_id uint64, next string) error {
	res, err := db.Exec(updateBlockNext, block_db_id, next)
	if err != nil {
		return err
	}
	numRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if numRows != 1 {
		return fmt.Errorf("UpdateBlockNext failed to update exactly 1 row (%d)", numRows)
	}
	return nil
}

func InsertVout(db *sql.DB, dbVout *dbtypes.Vout) (uint64, error) {
	insertStatement := MakeVoutInsertStatement(dbVout)
	var id uint64
	err := db.QueryRow(insertStatement,
		dbVout.Outpoint, dbVout.Value, dbVout.Ind, dbVout.Version,
		dbVout.ScriptPubKey, dbVout.ScriptPubKeyData.ReqSigs,
		dbVout.ScriptPubKeyData.Type).Scan(&id)
	return id, err
}

func InsertVouts(db *sql.DB, dbVouts []*dbtypes.Vout) ([]uint64, error) {
	dbtx, err := db.Begin()
	if err != nil {
		dbtx.Rollback()
		return nil, fmt.Errorf("Unable to begin database transaction: %v", err)
	}

	if _, err = dbtx.Exec("SET LOCAL synchronous_commit TO OFF;"); err != nil {
		dbtx.Rollback()
		return nil, err
	}

	ids := make([]uint64, len(dbVouts))
	for i, vout := range dbVouts {
		insertStatement := MakeVoutInsertStatement(vout)
		var id uint64
		err := db.QueryRow(insertStatement,
			vout.Outpoint, vout.Value, vout.Ind, vout.Version,
			vout.ScriptPubKey, vout.ScriptPubKeyData.ReqSigs,
			vout.ScriptPubKeyData.Type).Scan(&id)
		if err != nil {
			dbtx.Rollback()
			return nil, err
		}
		ids[i] = id
	}

	dbtx.Commit()
	return ids, nil
}

func InsertTx(db *sql.DB, dbTx *dbtypes.Tx) (uint64, error) {
	insertStatement := MakeTxInsertStatement(dbTx)
	var id uint64
	err := db.QueryRow(insertStatement,
		dbTx.BlockHash, dbTx.BlockIndex, dbTx.TxID,
		dbTx.Version, dbTx.Locktime, dbTx.Expiry,
		dbTx.NumVin, dbtypes.VinTxPropertyARRAY(dbTx.Vin),
		dbTx.NumVout).Scan(&id)
	return id, err
}
