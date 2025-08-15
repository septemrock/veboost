package contracts

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/Bedrock-Technology/VeMerkle/abi/airdrop"
	"github.com/Bedrock-Technology/VeMerkle/internal/config"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

var (
	instance *Proxy
	once     sync.Once
)

type Proxy struct {
	airdrop *airdrop.Airdrop
}

func InitProxy() error {
	var err error
	once.Do(func() {
		client, dialErr := ethclient.Dial(config.GetConfig().Contracts.Airdrop.RPC)
		if dialErr != nil {
			err = fmt.Errorf("failed to connect to ethereum client: %v", dialErr)
			return
		}

		address := common.HexToAddress(config.GetConfig().Contracts.Airdrop.Address)
		contract, contractErr := airdrop.NewAirdrop(address, client)
		if contractErr != nil {
			err = fmt.Errorf("failed to instantiate contract: %v", contractErr)
			return
		}

		instance = &Proxy{
			airdrop: contract,
		}
	})
	return err
}

func GetProxy() *Proxy {
	if instance == nil {
		panic("proxy not initialized")
	}
	return instance
}

func (p *Proxy) CheckEpochValidity(epoch uint64) (bool, error) {
	currentEpoch, err := p.airdrop.CurrentEpoch(&bind.CallOpts{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"epoch": epoch,
		}).Errorf("failed to get current epoch: %v", err)
		return false, err
	}
	nextEpoch := currentEpoch.Uint64() + 1
	if epoch != nextEpoch {
		return false, nil
	}
	return true, nil
}

func (p *Proxy) CheckCurEpochValidity(epoch uint64) (bool, error) {
	currentEpoch, err := p.airdrop.CurrentEpoch(&bind.CallOpts{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"epoch": epoch,
		}).Errorf("failed to get current epoch: %v", err)
		return false, err
	}
	if epoch != currentEpoch.Uint64() {
		return false, nil
	}
	return true, nil
}

func (p *Proxy) IsCurrentEpochActive() (bool, error) {
	active, err := p.airdrop.IsActive(&bind.CallOpts{})
	if err != nil {
		logrus.Errorf("Failed to check if current epoch is active: %v", err)
		return false, err
	}
	return active, nil
}

func (p *Proxy) GetCurrentEpoch() (uint64, error) {
	currentEpoch, err := p.airdrop.CurrentEpoch(&bind.CallOpts{})
	if err != nil {
		logrus.Errorf("Failed to get current epoch: %v", err)
		return 0, err
	}
	return currentEpoch.Uint64(), nil
}

func (p *Proxy) HasUsersClaimed(epoch *big.Int, users []common.Address) ([]bool, error) {
	claimedStatus, err := p.airdrop.HasClaimed(&bind.CallOpts{}, epoch, users)
	if err != nil {
		logrus.Errorf("Failed to check users claimed status: %v", err)
		return nil, err
	}
	return claimedStatus, nil
}
