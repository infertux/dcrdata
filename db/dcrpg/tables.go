package dcrpg

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	// Start the PostgreSQL driver
	_ "github.com/lib/pq"
)

//var tables = []string{"blocks", "transactions", "vouts"}
var tables = map[string]string{
	"blocks":       createBlockTable,
	"transactions": createTransactionTable,
	"vouts":        createVoutTable,
}

const (
	createBlockTable = `CREATE TABLE blocks (  
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

	createVinType = `CREATE TYPE vin AS (
		prev_tx_hash TEXT,
		prev_tx_index INTEGER,
		prev_tx_tree SMALLINT,
		sequence INTEGER,
		value_in DOUBLE PRECISION,
		block_height INT4,
		block_index INT4,
		script_hex BYTEA
	);`

	createTransactionTable = `CREATE TABLE transactions (
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

	createVoutTable = `CREATE TABLE vouts (
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

func TableExists(db *sql.DB, tableName string) (bool, error) {
	rows, err := db.Query(`select relname from pg_class where relname = $1`,
		tableName)
	defer rows.Close()
	return rows.Next(), err
}

func Connect(host, port, user, pass, dbname string) (*sql.DB, error) {
	var psqlInfo string
	if pass == "" {
		psqlInfo = fmt.Sprintf("host=%s port=%s user=%s "+
			"dbname=%s sslmode=disable",
			host, port, user, dbname)
	} else {
		psqlInfo = fmt.Sprintf("host=%s port=%s user=%s "+
			"password=%s dbname=%s sslmode=disable",
			host, port, user, pass, dbname)
	}

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	//defer db.Close()

	err = db.Ping()
	return db, err
}

func DropTables(db *sql.DB) {
	for tableName := range tables {
		fmt.Printf("DROPPING the \"%s\" table.\n", tableName)
		if err := dropTable(db, tableName); err != nil {
			fmt.Println(err)
		}
	}

	_, err := db.Exec(`DROP TYPE IF EXISTS vin;`)
	if err != nil {
		fmt.Println(err)
	}
}

func dropTable(db *sql.DB, tableName string) error {
	_, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s;`, tableName))
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func CreateTables(db *sql.DB) error {
	var err error
	for tableName, createCommand := range tables {
		var exists bool
		exists, err = TableExists(db, tableName)
		if err != nil {
			return err
		}
		fmt.Printf("Does the \"%s\" table exist? %v\n", tableName, exists)
		if !exists {
			fmt.Printf("Creating the \"%s\" table.\n", tableName)
			_, err = db.Exec(createCommand)
			if err != nil {
				return err
			}
			_, err = db.Exec(fmt.Sprintf(`COMMENT ON TABLE %s
				IS 'v1';`, tableName))
			if err != nil {
				return err
			}
		}
	}
	return err
}

func TableVersions(db *sql.DB) map[string]int32 {
	versions := map[string]int32{}
	for tableName := range tables {
		Result := db.QueryRow(`select obj_description($1::regclass);`, tableName)
		var s string
		v := int(-1)
		if Result != nil {
			Result.Scan(&s)
			re := regexp.MustCompile(`^v(\d+)$`)
			subs := re.FindStringSubmatch(s)
			if len(subs) > 1 {
				var err error
				v, err = strconv.Atoi(subs[1])
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		versions[tableName] = int32(v)
	}
	return versions
}
