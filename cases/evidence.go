package cases

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xbench/lib"
	"github.com/xuperchain/xuper-sdk-go/v2/account"
	"github.com/xuperchain/xuperchain/service/pb"
)

type evidence struct {
	host        string
	concurrency int
	length      int

	accounts []*account.Account
	encoders []*json.Encoder
	output   string
	sample   int
	txindex  []int
}

func NewEvidence(config *Config) (Generator, error) {
	inputSample, _ := strconv.Atoi(config.Args["sample"])
	t := &evidence{
		host:        config.Host,
		concurrency: config.Concurrency,
		output:      config.Args["output"],
		sample:      inputSample,
	}

	var err error
	t.length, err = strconv.Atoi(config.Args["length"])
	if err != nil {
		return nil, fmt.Errorf("evidence length error: %v", err)
	}

	t.accounts, err = lib.LoadAccount(t.concurrency)
	if err != nil {
		return nil, fmt.Errorf("load account error: %v", err)
	}

	t.encoders = make([]*json.Encoder, t.concurrency)
	for i := 0; i < t.concurrency; i++ {
		filename := fmt.Sprintf("evidence.dat.%04d", i)
		file, err := os.Create(filepath.Join(t.output, filename))
		if err != nil {
			return nil, fmt.Errorf("open output file error: %v", err)
		}
		t.encoders[i] = json.NewEncoder(file)
	}
	t.txindex = make([]int, t.concurrency)
	log.Printf("generate: type=evidence, concurrency=%d, length=%d", t.concurrency, t.length)
	return t, nil
}

func (t *evidence) Init() error {
	return nil
}

func (t *evidence) Generate(id int) (proto.Message, error) {
	ak := t.accounts[id]
	tx := EvidenceTx(ak, t.length)
	// sample等于0 表示不采样
	if t.sample != 0 {
		t.txindex[id]++
		if t.txindex[id] == t.sample {
			if err := t.encoders[id].Encode(tx); err != nil {
				log.Fatalf("write tx error: %v", err)
				return nil, err
			}
			t.txindex[id] = 0
		}
	}

	return tx, nil
}

func EvidenceTx(ak *account.Account, length int) *pb.Transaction {
	tx := &pb.Transaction{
		Version:   3,
		Desc:      lib.RandBytes(length),
		Nonce:     strconv.FormatInt(time.Now().UnixNano(), 36),
		Timestamp: time.Now().UnixNano(),
		Initiator: ak.Address,
	}

	lib.SignTx(tx, ak)
	return tx
}

func init() {
	RegisterGenerator(CaseEvidence, NewEvidence)
}
