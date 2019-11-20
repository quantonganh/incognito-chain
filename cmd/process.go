package main

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/privacy"
	"log"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
)

func parseToJsonString(data interface{}) ([]byte, error) {
	result, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, nil
}

func processCmd() {
	switch cfg.Command {
	case getPrivacyTokenID:
		{
			log.Printf("Params %+v", cfg)
			if cfg.PNetwork == "" {
				log.Println("Wrong param")
				return
			}
			if cfg.PToken == "" {
				log.Println("Wrong param")
				return
			}

			//hashPNetWork := common.HashH([]byte(cfg.PNetwork))
			//log.Printf("hashPNetWork: %+v\n", hashPNetWork.String())
			//copy(tokenID[:16], hashPNetWork[:16])
			//log.Printf("tokenID: %+v\n", tokenID.String())
			//
			//hashPToken := common.HashH([]byte(cfg.PToken))
			//log.Printf("hashPToken: %+v\n", hashPToken.String())
			//copy(tokenID[16:], hashPToken[:16])

			point := privacy.HashToPoint([]byte(cfg.PNetwork + "-" + cfg.PToken))
			hash := new(common.Hash)
			err := hash.SetBytes(point.ToBytesS())
			if err != nil {
				log.Println("Wrong param")
				return
			}

			log.Printf("Result tokenID: %+v\n", hash.String())
		}
	case createWalletCmd:
		{
			if cfg.WalletPassphrase == "" || cfg.WalletName == "" {
				log.Println("Wrong param")
				return
			}
			err := createWallet()
			if err != nil {
				log.Println(err)
				return
			}
		}
	case listWalletAccountCmd:
		{
			if cfg.WalletPassphrase == "" || cfg.WalletName == "" {
				log.Println("Wrong param")
				return
			}
			accounts, err := listAccounts()
			if err != nil {
				log.Println(err)
				return
			}
			result, err := parseToJsonString(accounts)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(string(result))
		}
	case getWalletAccountCmd:
		{
			if cfg.WalletPassphrase == "" || cfg.WalletName == "" || cfg.WalletAccountName == "" {
				log.Println("Wrong param")
				return
			}
			account, err := getAccount(cfg.WalletAccountName)
			if err != nil {
				log.Println(err)
				return
			}
			result, err := parseToJsonString(account)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(string(result))
		}
	case createWalletAccountCmd:
		{
			if cfg.WalletPassphrase == "" || cfg.WalletName == "" || cfg.WalletAccountName == "" {
				log.Println("Wrong param")
				return
			}
			var shardID *byte
			if cfg.ShardID > -1 {
				temp := byte(cfg.ShardID)
				shardID = &temp
			}
			account, err := createAccount(cfg.WalletAccountName, shardID)
			if err != nil {
				log.Println(err)
				return
			}
			result, err := parseToJsonString(account)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(string(result))
		}
	case backupChain:
		{
			if cfg.Beacon == false && cfg.ShardIDs == "" {
				log.Println("No Expected Params")
				return
			}
			bc, err := makeBlockChain(cfg.ChainDataDir, cfg.TestNet)
			if err != nil {
				log.Println("Error create blockchain variable ", err)
				return
			}
			if cfg.Beacon {
				err := backupBeaconChain(bc, cfg.OutDataDir, cfg.FileName)
				if err != nil {
					log.Printf("Beacon Beackup failed, err %+v", err)
				}
			}
			var shardIDs = []byte{}
			if cfg.ShardIDs != "" {
				// all shard
				if cfg.ShardIDs == "all" {
					var numberOfShards int
					if cfg.TestNet {
						numberOfShards = blockchain.ChainTestParam.ActiveShards
					} else {
						numberOfShards = blockchain.ChainMainParam.ActiveShards
					}
					for i := 0; i < numberOfShards; i++ {
						shardIDs = append(shardIDs, byte(i))
					}
				} else {
					// some particular shard
					strs := strings.Split(cfg.ShardIDs, ",")
					if len(strs) > 256 {
						log.Println("Number of shard id to process exceed limit")
						return
					}
					for _, value := range strs {
						temp, err := strconv.Atoi(value)
						if err != nil {
							log.Println("ShardID Params MUST contain number only in range 0-255")
							return
						}
						if temp > 256 {
							log.Println("ShardID exceed MAX value (> 255)")
							return
						}
						shardID := byte(temp)
						if common.IndexOfByte(shardID, shardIDs) > 0 {
							continue
						}
						shardIDs = append(shardIDs, shardID)
					}
				}
				//backup shard
				for _, shardID := range shardIDs {
					err := backupShardChain(bc, shardID, cfg.OutDataDir, cfg.FileName)
					if err != nil {
						log.Printf("Shard %+v back up failed, err %+v", shardID, err)
					}
				}
			}
		}
	case restoreChain:
		{
			if cfg.FileName == "" {
				log.Println("No Backup File to Process")
				return
			}
			bc, err := makeBlockChain(cfg.ChainDataDir, cfg.TestNet)
			if err != nil {
				log.Println("Error create blockchain variable ", err)
				return
			}
			if cfg.Beacon {
				err := restoreBeaconChain(bc, cfg.FileName)
				if err != nil {
					log.Printf("Beacon Restore failed, err %+v", err)
				}
			} else {
				filenames := strings.Split(cfg.FileName, ",")
				for _, filename := range filenames {
					err := RestoreShardChain(bc, filename)
					if err != nil {
						log.Printf("File %+v back up failed, err %+v", filename, err)
					}
				}
			}
		}
	}
}
