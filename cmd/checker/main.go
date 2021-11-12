package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/xuperchain/xuper-sdk-go/v2/xuper"
	"github.com/xuperchain/xuperchain/service/pb"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Path        string `yaml:"path"`
	Host        string `yaml:"host"`
	Concurrency int    `yaml:"concurrency"`
	Benchmark   string `yaml:"benchmark"`
}

type Checker struct {
	client    *xuper.XClient
	path      string
	files     []os.FileInfo
	benchmark string
}

func (c *Config) GetConf(confFile string) {

	yamlFile, err := ioutil.ReadFile(confFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func (checker *Checker) CheckTx(id int, ch chan int) {
	filename := checker.files[id].Name()
	file, err := os.Open(filepath.Join(checker.path, filename))
	decoder := json.NewDecoder(file)
	errTx := 0
	totalTx := 0
	for {
		var tx pb.Transaction
		err = decoder.Decode(&tx)
		if err == io.EOF {
			log.Printf("====file finish: %s total:%d err:%d=====", filename, totalTx, errTx)
			break
		}
		if err != nil {
			log.Printf("read tx file error: %v", err)
			break
		}
		totalTx++
		chainTx, err := checker.client.QueryTxByID(hex.EncodeToString(tx.Txid))
		if err != nil {
			log.Printf("query tx error: %v", err)
			continue
		}
		var x []byte
		var y []byte
		if checker.benchmark == "evidence" {
			x = tx.Desc
			y = chainTx.Desc
		} else if checker.benchmark == "short_content" {
			x = tx.ContractRequests[0].Args["content"]
			y = chainTx.ContractRequests[0].Args["content"]
		} else {
			log.Fatalf("unknow benchmark")
		}

		if !bytes.Equal(x, y) {
			errTx++
			log.Printf("diff desc txid:%v desc:%v desc_on_chain:%v", hex.EncodeToString(tx.Txid), string(x), string(y))
		} else {
			log.Printf("same content txid:%v desc:%v desc_on_chain:%v", hex.EncodeToString(tx.Txid), string(x), string(y))
		}
	}
	ch <- 1
}
func main() {
	var c Config
	c.GetConf("conf/checker.yaml")
	log.Printf("config: %v", c)

	files, err := ioutil.ReadDir(c.Path)
	if err != nil {
		log.Fatalf("check read dir error: %v", err)
	}

	if len(files) < c.Concurrency {
		log.Fatalf("file number less than concurrency")
	}

	client, err := xuper.New(c.Host)
	checker := &Checker{
		client:    client,
		path:      c.Path,
		files:     files,
		benchmark: c.Benchmark,
	}

	chs := make([]chan int, c.Concurrency)
	for i := 0; i < c.Concurrency; i++ {
		chs[i] = make(chan int)
		go checker.CheckTx(i, chs[i])
	}

	for _, ch := range chs {
		<-ch
	}
}
