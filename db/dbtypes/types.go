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
	txDbID           int64
	Value            float64          `json:"value"`
	N                uint32           `json:"n"`
	Version          uint16           `json:"version"`
	ScriptPubKey     string           `json:"pkScriptHex"`
	ScriptPubKeyData ScriptPubKeyData `json:"pkScript"`
}

// ScriptPubKeyData is part of the result of decodescript(ScriptPubKeyHex)
type ScriptPubKeyData struct {
	ReqSigs   int32    `json:"reqSigs"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses"`
	CommitAmt float64  `json:"commitamt"`
}

type VinTxProperty struct {
	Coinbase    string  `json:"coinbase"`
	PrevTxHash  string  `json:"prevtxhash"`
	PrevTxIndex uint32  `json:"prevvoutidx"`
	Tree        int8    `json:"tree"`
	Sequence    uint32  `json:"sequence"`
	AmountIn    float64 `json:"amountin"`
	ScriptHex   string  `json:"scripthex"`
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
	Txid       string `json:"txid"`
	Version    int32  `json:"version"`
	Locktime   uint32 `json:"locktime"`
	Expiry     uint32 `json:"expiry"`
	NumVin     uint32 `json:"numvin"`
	Vin        []Vin  `json:"vin"`
	VoutDbIds  []int64
	// NOTE: VoutDbIds may not be needed if there is a vout table since each
	// vout will have a tx_dbid
}

type Block struct {
	Hash         string `json:"hash"`
	Size         int32  `json:"size"`
	Height       int64  `json:"height"`
	Version      int32  `json:"version"`
	MerkleRoot   string `json:"merkleroot"`
	StakeRoot    string `json:"stakeroot"`
	txDbIDs      []int64
	Tx           []string `json:"tx"`
	STx          []string `json:"stx"`
	Time         int64    `json:"time"`
	Nonce        uint32   `json:"nonce"`
	VoteBits     uint16   `json:"votebits"`
	FinalState   string   `json:"finalstate"`
	Voters       uint16   `json:"voters"`
	FreshStake   uint8    `json:"freshstake"`
	Revocations  uint8    `json:"revocations"`
	PoolSize     uint32   `json:"poolsize"`
	Bits         string   `json:"bits"`
	SBits        float64  `json:"sbits"`
	Difficulty   float64  `json:"difficulty"`
	ExtraData    string   `json:"extradata"`
	StakeVersion uint32   `json:"stakeversion"`
	PreviousHash string   `json:"previousblockhash"`
	NextHash     string   `json:"nextblockhash"`
}
