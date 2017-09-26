package internal

import "fmt"

const (
	insertBlockRow = `INSERT INTO blocks (
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

	insertTxRow = `INSERT INTO transactions (
		block_hash, block_index, tx_hash, version,
		lock_time, expiry, num_vin, vins,
		num_vout, vout_db_ids) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, %s) RETURNING id;`

	insertVoutRow = `INSERT INTO vouts (outpoint, value, ind, version,
		pkscript, script_req_sigs, script_type, script_addresses)
		VALUES ($1, $2, $3, $4, $5, $6, $7, %s) RETURNING id;`

	CreateBlockTable = `CREATE TABLE blocks (  
		id SERIAL PRIMARY KEY,
		hash TEXT UNIQUE NOT NULL,
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

	CreateVinType = `CREATE TYPE vin AS (
		prev_tx_hash TEXT,
		prev_tx_index INTEGER,
		prev_tx_tree SMALLINT,
		sequence INTEGER,
		value_in DOUBLE PRECISION,
		block_height INT4,
		block_index INT4,
		script_hex BYTEA
	);`

	CreateTransactionTable = `CREATE TABLE transactions (
		id SERIAL8 PRIMARY KEY,
		/*block_db_id INT4,*/
		block_hash TEXT,
		block_index INT4,
		tx_hash TEXT UNIQUE,
		version INT4,
		lock_time INT4,
		expiry INT4,
		num_vin INT4,
		vins JSONB,
		num_vout INT4,
		vout_db_ids INT8[]
	);`

	CreateVoutTable = `CREATE TABLE vouts (
		id SERIAL8 PRIMARY KEY,
		/* tx_db_id INT8, */
		outpoint TEXT UNIQUE,
		value INT8,
		ind INT4,
		version INT2,
		pkscript BYTEA,
		script_req_sigs INT4,
		script_type TEXT,
		script_addresses TEXT[]
	);`
)

func MakeBlockInsertStatement(txDbIDs, stxDbIDs []uint64, rtxs, stxs []string) string {
	rtxDbIDsARRAY := makeARRAYOfBIGINTs(txDbIDs)
	stxDbIDsARRAY := makeARRAYOfBIGINTs(stxDbIDs)
	rtxTEXTARRAY := makeARRAYOfTEXT(rtxs)
	stxTEXTARRAY := makeARRAYOfTEXT(stxs)
	return fmt.Sprintf(insertBlockRow, rtxTEXTARRAY, rtxDbIDsARRAY,
		stxTEXTARRAY, stxDbIDsARRAY)
}

func MakeVoutInsertStatement(scriptAddresses []string) string {
	addrs := makeARRAYOfTEXT(scriptAddresses)
	return fmt.Sprintf(insertVoutRow, addrs)
}

func MakeTxInsertStatement(voutDbIDs []uint64) string {
	dbIDsBIGINT := makeARRAYOfBIGINTs(voutDbIDs)
	return fmt.Sprintf(insertTxRow, dbIDsBIGINT)
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
