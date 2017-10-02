package internal

import "fmt"

const (
	// Block insert
	insertBlockRow0 = `INSERT INTO blocks (
		hash, height, size, version, merkle_root, stake_root,
		numtx, num_rtx, tx, txDbIDs, num_stx, stx, stxDbIDs,
		time, nonce, vote_bits, final_state, voters,
		fresh_stake, revocations, pool_size, bits, sbits, 
		difficulty, extra_data, stake_version, previous_hash)
	VALUES ($1, $2, $3, $4, $5, $6, 
		$7, $8, %s, %s, $9, %s, %s,
		$10, $11, $12, $13, $14, 
		$15, $16, $17, $18, $19, 
		$20, $21, $22, $23) `
	insertBlockRow         = insertBlockRow0 + `RETURNING id;`
	insertBlockRowChecked  = insertBlockRow0 + `ON CONFLICT (hash) DO NOTHING RETURNING id;`
	insertBlockRowReturnId = `WITH ins AS (` +
		insertBlockRow0 +
		`ON CONFLICT (hash) DO UPDATE
		SET hash = NULL WHERE FALSE
		RETURNING id
		)
	SELECT id FROM ins
	UNION  ALL
	SELECT id FROM blocks
	WHERE  hash = $1
	LIMIT  1;`

	// Transaction insert
	insertTxRow0 = `INSERT INTO transactions (
		block_hash, block_index, tree, tx_hash, version,
		lock_time, expiry, num_vin, vins, vin_db_ids,
		num_vout, vout_db_ids)
	VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8, $9, %s,
		$10, %s) `
	insertTxRow        = insertTxRow0 + `RETURNING id;`
	insertTxRowChecked = insertTxRow0 + `ON CONFLICT (tx_hash, block_hash) DO NOTHING RETURNING id;`
	upsertTxRow        = insertTxRow0 + `ON CONFLICT (tx_hash, block_hash) DO UPDATE 
		SET block_hash = $1, block_index = $2, tree = $3 RETURNING id;`
	insertTxRowReturnId = `WITH ins AS (` +
		insertTxRow0 +
		`ON CONFLICT (tx_hash, block_hash) DO UPDATE
		SET tx_hash = NULL WHERE FALSE
		RETURNING id
		)
	SELECT id FROM ins
	UNION  ALL
	SELECT id FROM blocks
	WHERE  tx_hash = $3 AND block_hash = $1
	LIMIT  1;`

	insertVoutRow0 = `INSERT INTO vouts (outpoint, value, ind, version,
		pkscript, script_req_sigs, script_type, script_addresses)
	VALUES ($1, $2, $3, $4, $5, $6, $7, %s) `
	insertVoutRow         = insertVoutRow0 + `RETURNING id;`
	insertVoutRowChecked  = insertVoutRow0 + `ON CONFLICT (outpoint) DO NOTHING RETURNING id;`
	insertVoutRowReturnId = `WITH ins AS (` +
		insertVoutRow0 +
		`ON CONFLICT (outpoint) DO UPDATE
		SET outpoint = NULL WHERE FALSE
		RETURNING id
		)
	 SELECT id FROM ins
	 UNION  ALL
	 SELECT id FROM vouts
	 WHERE  outpoint = $1
	 LIMIT  1;`

	CreateBlockTable = `CREATE TABLE IF NOT EXISTS blocks (  
		id SERIAL PRIMARY KEY,
		hash TEXT NOT NULL, -- UNIQUE
		height INT4,
		size INT4,
		version INT4,
		merkle_root TEXT,
		stake_root TEXT,
		numtx INT4,
		num_rtx INT4,
		tx TEXT[],
		txDbIDs INT8[],
		num_stx INT4,
		stx TEXT[],
		stxDbIDs INT8[],
		time INT8,
		nonce INT8,
		vote_bits INT2,
		final_state BYTEA,
		voters INT2,
		fresh_stake INT2,
		revocations INT2,
		pool_size INT4,
		bits INT4,
		sbits INT8,
		difficulty FLOAT8,
		extra_data BYTEA,
		stake_version INT4,
		previous_hash TEXT,
		next_hash TEXT
	);`

	IndexBlockTableOnHash = `CREATE UNIQUE INDEX uix_block_hash
		ON blocks(hash);`
	DeindexBlockTableOnHash = `DROP INDEX uix_block_hash;`

	RetrieveBestBlock       = `SELECT * FROM blocks ORDER BY height DESC LIMIT 0, 1;`
	RetrieveBestBlockHeight = `SELECT id, hash, height FROM blocks ORDER BY height DESC LIMIT 1;`

	CreateVinType = `CREATE TYPE vin AS (
		prev_out TEXT,
		prev_tx_hash TEXT,
		prev_tx_index INTEGER,
		prev_tx_tree SMALLINT,
		sequence INTEGER,
		value_in DOUBLE PRECISION,
		block_height INT4,
		block_index INT4,
		script_hex BYTEA
	);`

	CreateVinTable = `CREATE TABLE IF NOT EXISTS vins (
		id SERIAL8 PRIMARY KEY,
		tx_hash TEXT,
		tx_index INT4,
		prev_tx_hash TEXT,
		prev_tx_index INT8
	);`

	InsertVinRow0 = `INSERT INTO vins (tx_hash, tx_index, prev_tx_hash, prev_tx_index)
		VALUES ($1, $2, $3, $4) `
	InsertVinRow        = InsertVinRow0 + `RETURNING id;`
	InsertVinRowChecked = InsertVinRow0 +
		`ON CONFLICT (tx_hash, tx_index) DO NOTHING RETURNING id;`

	IndexVinTableOnVins = `CREATE INDEX uix_vin
		ON vins(tx_hash, tx_index)
		;` // STORING (prev_tx_hash, prev_tx_index)
	IndexVinTableOnPrevOuts = `CREATE INDEX uix_vin_prevout
		ON vins(prev_tx_hash, prev_tx_index)
		;` // STORING (tx_hash, tx_index)
	DeindexVinTableOnVins     = `DROP INDEX uix_vin;`
	DeindexVinTableOnPrevOuts = `DROP INDEX uix_vin_prevout;`

	CreateTransactionTable = `CREATE TABLE IF NOT EXISTS transactions (
		id SERIAL8 PRIMARY KEY,
		/*block_db_id INT4,*/
		block_hash TEXT,
		block_index INT4,
		tree INT2,
		tx_hash TEXT,
		version INT4,
		lock_time INT4,
		expiry INT4,
		num_vin INT4,
		vins JSONB,
		vin_db_ids INT8[],
		num_vout INT4,
		vout_db_ids INT8[]
	);`

	IndexTransactionTableOnBlockIn = `CREATE UNIQUE INDEX uix_tx_block_in
		ON transactions(block_hash, block_index, tree)
		;` // STORING (tx_hash, block_hash)
	DeindexTransactionTableOnBlockIn = `DROP INDEX uix_tx_block_in;`

	IndexTransactionTableOnHashes = `CREATE UNIQUE INDEX uix_tx_hashes
		 ON transactions(tx_hash, block_hash)
		 ;` // STORING (block_hash, block_index, tree)
	DeindexTransactionTableOnHashes = `DROP INDEX uix_tx_hashes;`

	SelectTxByPrevOut = `SELECT * FROM transactions WHERE vins @> json_build_array(json_build_object('prevtxhash',$1)::jsonb)::jsonb;`
	//SelectTxByPrevOut = `SELECT * FROM transactions WHERE vins #>> '{"prevtxhash"}' = '$1';`

	SelectTxsByPrevOutTx = `SELECT * FROM transactions WHERE vins @> json_build_array(json_build_object('prevtxhash',$1::TEXT)::jsonb)::jsonb;`
	// '[{"prevtxhash":$1}]'

	SelectSpendingTxsByPrevTx = `SELECT id, tx_hash FROM vins WHERE prev_tx_hash=$1;`
	SelectSpendingTxByPrevOut = `SELECT id, tx_hash FROM vins WHERE prev_tx_hash=$1 AND prev_tx_index=$2;`
	SelectFundingTxsByTx      = `SELECT id, prev_tx_hash FROM vins WHERE tx_hash=$1;`
	SelectFundingTxByTxIn     = `SELECT id, prev_tx_hash FROM vins WHERE tx_hash=$1 AND tx_index=$2;`

	CreateVoutTable = `CREATE TABLE IF NOT EXISTS vouts (
		id SERIAL8 PRIMARY KEY,
		/* tx_db_id INT8, */
		outpoint TEXT, -- UNIQUE
		value INT8,
		ind INT4,
		version INT2,
		pkscript BYTEA,
		script_req_sigs INT4,
		script_type TEXT,
		script_addresses TEXT[]
	);`

	SelectVoutByID = `SELECT * FROM vouts WHERE id=$1;`

	CreateBlockPrevNextTable = `CREATE TABLE IF NOT EXISTS block_chain (
		block_db_id INT8 PRIMARY KEY,
		prev_hash TEXT NOT NULL,
		this_hash TEXT UNIQUE NOT NULL, -- UNIQUE
		next_hash TEXT
	);`
)

func MakeBlockInsertStatement(txDbIDs, stxDbIDs []uint64, rtxs, stxs []string, checked bool) string {
	rtxDbIDsARRAY := makeARRAYOfBIGINTs(txDbIDs)
	stxDbIDsARRAY := makeARRAYOfBIGINTs(stxDbIDs)
	rtxTEXTARRAY := makeARRAYOfTEXT(rtxs)
	stxTEXTARRAY := makeARRAYOfTEXT(stxs)
	var insert string
	if checked {
		insert = insertBlockRowChecked
	} else {
		insert = insertBlockRow
	}
	return fmt.Sprintf(insert, rtxTEXTARRAY, rtxDbIDsARRAY,
		stxTEXTARRAY, stxDbIDsARRAY)
}

func MakeVoutInsertStatement(scriptAddresses []string, checked bool) string {
	addrs := makeARRAYOfTEXT(scriptAddresses)
	var insert string
	if checked {
		insert = insertVoutRowChecked
	} else {
		insert = insertVoutRow
	}
	return fmt.Sprintf(insert, addrs)
}

func MakeTxInsertStatement(voutDbIDs, vinDbIDs []uint64, checked bool) string {
	voutDbIDsBIGINT := makeARRAYOfBIGINTs(voutDbIDs)
	vinDbIDsBIGINT := makeARRAYOfBIGINTs(vinDbIDs)
	var insert string
	if checked {
		insert = insertTxRowChecked
	} else {
		insert = insertTxRow
	}
	return fmt.Sprintf(insert, voutDbIDsBIGINT, vinDbIDsBIGINT)
}

func makeARRAYOfTEXT(text []string) string {
	if len(text) == 0 {
		return "ARRAY['']"
	}
	sTEXTARRAY := "ARRAY["
	for i, txt := range text {
		if i == len(text)-1 {
			sTEXTARRAY += fmt.Sprintf(`'%s'`, txt)
			break
		}
		sTEXTARRAY += fmt.Sprintf(`'%s', `, txt)
	}
	sTEXTARRAY += "]"
	return sTEXTARRAY
}

func makeARRAYOfBIGINTs(ints []uint64) string {
	if len(ints) == 0 {
		return "ARRAY[]::BIGINT[]"
	}
	ARRAY := "ARRAY["
	for i, v := range ints {
		if i == len(ints)-1 {
			ARRAY += fmt.Sprintf(`%d`, v)
			break
		}
		ARRAY += fmt.Sprintf(`%d, `, v)
	}
	ARRAY += "]"
	return ARRAY
}
