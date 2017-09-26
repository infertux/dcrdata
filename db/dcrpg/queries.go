package dcrpg

import (
	"database/sql"

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

func InsertVout(db *sql.DB, dbVout *dbtypes.Vout) (uint64, error) {
	insertStatement := MakeVoutInsertStatement(dbVout)
	var id uint64
	err := db.QueryRow(insertStatement,
		dbVout.Outpoint, dbVout.Value, dbVout.Ind, dbVout.Version,
		dbVout.ScriptPubKey, dbVout.ScriptPubKeyData.ReqSigs,
		dbVout.ScriptPubKeyData.Type).Scan(&id)
	return id, err
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
