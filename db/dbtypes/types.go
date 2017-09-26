package dbtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONB is used to implement the sql.Scanner and driver.Valuer interfaces
// requried for the type to make a postgresql compatible JSONB type.
type JSONB map[string]interface{}

// Value satisfies driver.Valuer
func (p JSONB) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, err
}

// Scan satisfies sql.Scanner
func (p *JSONB) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Scan type assertion .([]byte) failed.")
	}

	var i interface{}
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}

	// Set this JSONB
	*p, ok = i.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Type assertion .(map[string]interface{}) failed.")
	}

	return nil
}

// Vout defines a transaction output
type Vout struct {
	// txDbID           int64
	Value            int64            `json:"value"`
	Ind              uint32           `json:"ind"`
	Version          uint16           `json:"version"`
	ScriptPubKey     []byte           `json:"pkScriptHex"`
	ScriptPubKeyData ScriptPubKeyData `json:"pkScript"`
}

// ScriptPubKeyData is part of the result of decodescript(ScriptPubKeyHex)
type ScriptPubKeyData struct {
	ReqSigs   uint32   `json:"reqSigs"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses"`
}

type VinTxProperty struct {
	PrevTxHash  string `json:"prevtxhash"`
	PrevTxIndex uint32 `json:"prevvoutidx"`
	PrevTxTree  uint16 `json:"tree"`
	Sequence    uint32 `json:"sequence"`
	ValueIn     int64  `json:"amountin"`
	BlockHeight uint32 `json:"blockheight"`
	BlockIndex  uint32 `json:"blockindex"`
	ScriptHex   []byte `json:"scripthex"`
}

type Vin struct {
	//txDbID      int64
	Coinbase    string  `json:"coinbase"`
	TxHash      string  `json:"txhash"`
	VoutIdx     uint32  `json:"voutidx"`
	Tree        int8    `json:"tree"`
	Sequence    uint32  `json:"sequence"`
	AmountIn    float64 `json:"amountin"`
	BlockHeight uint32  `json:"blockheight"`
	BlockIndex  uint32  `json:"blockindex"`
	ScriptHex   string  `json:"scripthex"`
}

// ScriptSig models the signature script used to redeem the origin transaction
// as a JSON object (non-coinbase txns only)
type ScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

type Tx struct {
	blockDbID  int64
	BlockHash  string
	BlockIndex uint32
	TxID       string          `json:"txid"`
	Version    uint16          `json:"version"`
	Locktime   uint32          `json:"locktime"`
	Expiry     uint32          `json:"expiry"`
	NumVin     uint32          `json:"numvin"`
	Vin        []VinTxProperty `json:"vin"`
	VoutDbIds  []int64
	// NOTE: VoutDbIds may not be needed if there is a vout table since each
	// vout will have a tx_dbid
}

type Block struct {
	Hash         string `json:"hash"`
	Size         uint32 `json:"size"`
	Height       uint32 `json:"height"`
	Version      uint32 `json:"version"`
	MerkleRoot   string `json:"merkleroot"`
	StakeRoot    string `json:"stakeroot"`
	NumTx        uint32
	TxDbIDs      []int64
	NumRegTx     uint32
	Tx           []string `json:"tx"`
	NumStakeTx   uint32
	STx          []string `json:"stx"`
	Time         uint32   `json:"time"`
	Nonce        uint32   `json:"nonce"`
	VoteBits     uint16   `json:"votebits"`
	FinalState   [6]byte  `json:"finalstate"`
	Voters       uint16   `json:"voters"`
	FreshStake   uint8    `json:"freshstake"`
	Revocations  uint8    `json:"revocations"`
	PoolSize     uint32   `json:"poolsize"`
	Bits         uint32   `json:"bits"`
	SBits        int64    `json:"sbits"`
	Difficulty   float64  `json:"difficulty"`
	ExtraData    [32]byte `json:"extradata"`
	StakeVersion uint32   `json:"stakeversion"`
	PreviousHash string   `json:"previousblockhash"`
	//NextHash     string   `json:"nextblockhash"`
}
