package vault

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vultisig/verifier/vault_config"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	keygen "github.com/vultisig/commondata/go/vultisig/keygen/v1"
	vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"github.com/vultisig/vultiserver/relay"
	vgrelay "github.com/vultisig/vultisig-go/relay"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
	vgtypes "github.com/vultisig/vultisig-go/types"
)

var TssKeyGenTimeout = errors.New("keygen timeout")

type DKLSTssService struct {
	cfg                vault_config.Config
	messenger          *relay.MessengerImp
	logger             *logrus.Logger
	localStateAccessor *LocalStateAccessorImp
	isKeygenFinished   *atomic.Bool
	isKeysignFinished  *atomic.Bool
	storage            Storage
	queueClient        *asynq.Client
}

func NewDKLSTssService(cfg vault_config.Config,
	storage Storage,
	queueClient *asynq.Client) (*DKLSTssService, error) {
	return &DKLSTssService{
		cfg:                cfg,
		logger:             logrus.WithField("service", "dkls").Logger,
		isKeygenFinished:   &atomic.Bool{},
		isKeysignFinished:  &atomic.Bool{},
		storage:            storage,
		localStateAccessor: NewLocalStateAccessorImp(nil),
		queueClient:        queueClient,
	}, nil
}

func (t *DKLSTssService) GetMPCKeygenWrapper(isEdDSA bool) *MPCWrapperImp {
	return NewMPCWrapperImp(isEdDSA)
}

func (t *DKLSTssService) ProcessDKLSKeygen(req vgtypes.VaultCreateRequest) (string, string, error) {
	serverURL := t.cfg.Relay.Server
	relayClient := vgrelay.NewRelayClient(serverURL)

	if err := EnforceKeygen()

	// Let's register session here
	if err := relayClient.RegisterSession(req.SessionID, req.LocalPartyId); err != nil {
		return "", "", fmt.Errorf("failed to register session: %w", err)
	}
	// wait longer for keygen start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	partiesJoined, err := relayClient.WaitForSessionStart(ctx, req.SessionID)
	if err != nil {
		return "", "", fmt.Errorf("failed to wait for session start: %w", err)
	}
	t.logger.WithFields(logrus.Fields{
		"sessionID":      req.SessionID,
		"parties_joined": partiesJoined,
	}).Info("Session started")

	// create ECDSA key
	publicKeyECDSA, chainCodeECDSA, err := t.keygenWithRetry(req.SessionID, req.HexEncryptionKey, req.LocalPartyId, false, partiesJoined)
	if err != nil {
		return "", "", fmt.Errorf("failed to keygen ECDSA: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	// create EdDSA key
	publicKeyEdDSA, _, err := t.keygenWithRetry(req.SessionID, req.HexEncryptionKey, req.LocalPartyId, true, partiesJoined)
	if err != nil {
		return "", "", fmt.Errorf("failed to keygen EdDSA: %w", err)
	}

	if err := relayClient.CompleteSession(req.SessionID, req.LocalPartyId); err != nil {
		t.logger.WithFields(logrus.Fields{
			"session": req.SessionID,
			"error":   err,
		}).Error("Failed to complete session")
	}

	if isCompleted, err := relayClient.CheckCompletedParties(req.SessionID, partiesJoined); err != nil || !isCompleted {
		t.logger.WithFields(logrus.Fields{
			"sessionID":   req.SessionID,
			"isCompleted": isCompleted,
			"error":       err,
		}).Error("Failed to check completed parties")
	}
	err = t.BackupVault(req.Name, req.LocalPartyId, req.Email, req.PluginID,
		partiesJoined, publicKeyECDSA, publicKeyEdDSA, chainCodeECDSA, t.localStateAccessor)
	if err != nil {
		return "", "", fmt.Errorf("failed to backup vault: %w", err)
	}
	return publicKeyECDSA, publicKeyEdDSA, nil
}

func (t *DKLSTssService) BackupVault(vaultName, localPartyId, email, pluginId string,
	partiesJoined []string,
	ecdsaPubkey, eddsaPubkey string,
	hexChainCode string,
	localStateAccessor *LocalStateAccessorImp) error {
	ecdsaKeyShare, err := localStateAccessor.GetLocalState(ecdsaPubkey)
	if err != nil {
		return fmt.Errorf("failed to get local sate: %w", err)
	}

	eddsaKeyShare, err := localStateAccessor.GetLocalState(eddsaPubkey)
	if err != nil {
		return fmt.Errorf("failed to get local sate: %w", err)
	}

	vault := &vaultType.Vault{
		Name:           vaultName,
		PublicKeyEcdsa: ecdsaPubkey,
		PublicKeyEddsa: eddsaPubkey,
		Signers:        partiesJoined,
		CreatedAt:      timestamppb.New(time.Now()),
		HexChainCode:   hexChainCode,
		KeyShares: []*vaultType.Vault_KeyShare{
			{
				PublicKey: ecdsaPubkey,
				Keyshare:  ecdsaKeyShare,
			},
			{
				PublicKey: eddsaPubkey,
				Keyshare:  eddsaKeyShare,
			},
		},
		LocalPartyId:  localPartyId,
		ResharePrefix: "",
		LibType:       keygen.LibType_LIB_TYPE_DKLS,
	}

	return t.SaveVaultToStorage(vault, email, pluginId)
}

func (t *DKLSTssService) SaveVaultToStorage(vault *vaultType.Vault, email, pluginId string) error {
	if len(t.cfg.EncryptionSecret) == 0 {
		return errors.New("encryption secret is empty")
	}

	if len(pluginId) == 0 {
		return errors.New("failed to save vault to storage,plugin id is empty")
	}

	vaultData, err := proto.Marshal(vault)
	if err != nil {
		return fmt.Errorf("failed to Marshal vault: %w", err)
	}

	vaultData, err = common.EncryptVault(t.cfg.EncryptionSecret, vaultData)
	if err != nil {
		return fmt.Errorf("common.EncryptVault failed: %w", err)
	}

	vaultBackup := &vaultType.VaultContainer{
		Version:     1,
		Vault:       base64.StdEncoding.EncodeToString(vaultData),
		IsEncrypted: true,
	}

	filePathName := common.GetVaultBackupFilename(vault.PublicKeyEcdsa, pluginId)
	vaultBackupData, err := proto.Marshal(vaultBackup)
	if err != nil {
		return fmt.Errorf("failed to Marshal vaultBackup: %w", err)
	}

	base64VaultContent := base64.StdEncoding.EncodeToString(vaultBackupData)

	if t.cfg.QueueEmailTask && len(email) != 0 {
		emailRequest := vtypes.EmailRequest{
			Email:       email,
			FileName:    common.GetVaultName(vault),
			FileContent: base64VaultContent,
			VaultName:   vault.Name,
		}
		buf, err := json.Marshal(emailRequest)
		if err != nil {
			return fmt.Errorf("json.Marshal failed: %w", err)
		}

		taskInfo, err := t.queueClient.Enqueue(asynq.NewTask(EmailVaultBackupTypeName, buf),
			asynq.Retention(10*time.Minute),
			asynq.Queue(EmailQueueName))
		if err != nil {
			t.logger.Errorf("fail to enqueue email task: %v", err)
		}
		t.logger.Info("Email task enqueued: ", taskInfo.ID)
	}
	return t.storage.SaveVault(filePathName, []byte(base64VaultContent))
}

func (t *DKLSTssService) keygenWithRetry(sessionID string,
	hexEncryptionKey string,
	localPartyID string,
	isEdDSA bool,
	keygenCommittee []string) (string, string, error) {
	for i := 0; i < 3; i++ {
		publicKey, chainCode, err := t.keygen(sessionID, hexEncryptionKey, localPartyID, isEdDSA, keygenCommittee, i)
		if err != nil {
			t.logger.WithFields(logrus.Fields{
				"session_id":       sessionID,
				"local_party_id":   localPartyID,
				"keygen_committee": keygenCommittee,
				"attempt":          i,
			}).Error(err)
			time.Sleep(50 * time.Millisecond)
			continue
		} else {
			return publicKey, chainCode, nil
		}
	}
	return "", "", errors.New("fail to keygen after max retry")
}

func (t *DKLSTssService) keygen(sessionID string,
	hexEncryptionKey string,
	localPartyID string,
	isEdDSA bool,
	keygenCommittee []string,
	attempt int) (string, string, error) {
	t.logger.WithFields(logrus.Fields{
		"session_id":       sessionID,
		"local_party_id":   localPartyID,
		"keygen_committee": keygenCommittee,
		"attempt":          attempt,
	}).Info("Keygen")
	t.isKeygenFinished.Store(false)
	relayClient := vgrelay.NewRelayClient(t.cfg.Relay.Server)
	mpcKeygenWrapper := t.GetMPCKeygenWrapper(isEdDSA)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	// retrieve the setup Message
	encryptedEncodedSetupMsg, err := relayClient.WaitForSetupMessage(ctx, sessionID, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to get setup message: %w", err)
	}
	setupMessageBytes, err := t.decodeDecryptMessage(encryptedEncodedSetupMsg, hexEncryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode setup message: %w", err)
	}

	handle, err := mpcKeygenWrapper.KeygenSessionFromSetup(setupMessageBytes, []byte(localPartyID))
	if err != nil {
		return "", "", fmt.Errorf("failed to create session from setup message: %w", err)
	}
	defer func() {
		if err := mpcKeygenWrapper.KeygenSessionFree(handle); err != nil {
			t.logger.Error("failed to free keygen session", "error", err)
		}
	}()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		if err := t.processKeygenOutbound(handle, sessionID, hexEncryptionKey, keygenCommittee, localPartyID, isEdDSA, wg); err != nil {
			t.logger.Error("failed to process keygen outbound", "error", err)
		}
	}()
	publicKey, chainCode, err := t.processKeygenInbound(handle, sessionID, hexEncryptionKey, isEdDSA, localPartyID, wg)
	wg.Wait()
	return publicKey, chainCode, err
}

func (t *DKLSTssService) processKeygenOutbound(handle Handle,
	sessionID string,
	hexEncryptionKey string,
	parties []string,
	localPartyID string,
	isEdDSA bool,
	wg *sync.WaitGroup) error {
	defer wg.Done()
	messenger := relay.NewMessenger(t.cfg.Relay.Server, sessionID, hexEncryptionKey, true, "")
	mpcKeygenWrapper := t.GetMPCKeygenWrapper(isEdDSA)
	var startTime *time.Time
	for {
		outbound, err := mpcKeygenWrapper.KeygenSessionOutputMessage(handle)
		if err != nil {
			t.logger.Error("failed to get output message", "error", err)
		}
		if len(outbound) == 0 {
			if t.isKeygenFinished.Load() {
				// even when the local party is finished , we better to wait for a little while to guarantee the outbound message is sent
				if startTime == nil {
					n := time.Now()
					startTime = &n
					continue
				}

				if time.Since(*startTime) < time.Second {
					continue
				}

				// we are finished
				return nil
			}
			time.Sleep(time.Millisecond * 100)
			continue
		}
		encodedOutbound := base64.StdEncoding.EncodeToString(outbound)
		for i := 0; i < len(parties); i++ {
			receiver, err := mpcKeygenWrapper.KeygenSessionMessageReceiver(handle, outbound, i)
			if err != nil {
				t.logger.Error("failed to get receiver message", "error", err)
			}
			if len(receiver) == 0 {
				continue
			}

			t.logger.Infoln("Sending message to", receiver)
			// send the message to the receiver
			if err := messenger.Send(localPartyID, receiver, encodedOutbound); err != nil {
				t.logger.Errorf("failed to send message: %v", err)
			}
		}
	}
}

func (t *DKLSTssService) processKeygenInbound(handle Handle,
	sessionID string,
	hexEncryptionKey string,
	isEdDSA bool,
	localPartyID string,
	wg *sync.WaitGroup) (string, string, error) {
	defer wg.Done()
	var messageCache sync.Map
	mpcKeygenWrapper := t.GetMPCKeygenWrapper(isEdDSA)
	relayClient := vgrelay.NewRelayClient(t.cfg.Relay.Server)
	start := time.Now()
	for {
		select {
		case <-time.After(time.Millisecond * 100):
			if time.Since(start) > (time.Minute * 2) { // 2 minute timeout
				t.isKeygenFinished.Store(true)
				t.logger.Error("keygen timeout")
				return "", "", TssKeyGenTimeout
			}
			messages, err := relayClient.DownloadMessages(sessionID, localPartyID, "")
			if err != nil {
				t.logger.Error("failed to download messages", "error", err)
				continue
			}
			for _, message := range messages {
				if message.From == localPartyID {
					continue
				}
				cacheKey := fmt.Sprintf("%s-%s-%s", sessionID, localPartyID, message.Hash)
				if _, found := messageCache.Load(cacheKey); found {
					t.logger.Infof("Message already applied, skipping,hash: %s", message.Hash)
					continue
				}

				inboundBody, err := t.decodeDecryptMessage(message.Body, hexEncryptionKey)
				if err != nil {
					t.logger.Error("fail to decode inbound message", "error", err)
					continue
				}
				t.logger.Infoln("Received message from", message.From)
				isFinished, err := mpcKeygenWrapper.KeygenSessionInputMessage(handle, inboundBody)
				if err != nil {
					t.logger.Error("fail to apply input message", "error", err)
					continue
				}

				if err := relayClient.DeleteMessageFromServer(sessionID, localPartyID, message.Hash, ""); err != nil {
					t.logger.Error("fail to delete message", "error", err)
				}
				if isFinished {
					t.logger.Infoln("Keygen finished")
					result, err := mpcKeygenWrapper.KeygenSessionFinish(handle)
					if err != nil {
						t.logger.Error("fail to finish keygen", "error", err)
						return "", "", err
					}
					buf, err := mpcKeygenWrapper.KeyshareToBytes(result)
					if err != nil {
						t.logger.Error("fail to convert keyshare to bytes", "error", err)
						return "", "", err
					}
					encodedShare := base64.StdEncoding.EncodeToString(buf)
					publicKeyECDSABytes, err := mpcKeygenWrapper.KeysharePublicKey(result)
					if err != nil {
						t.logger.Error("fail to get public key", "error", err)
						return "", "", err
					}
					encodedPublicKey := hex.EncodeToString(publicKeyECDSABytes)
					t.logger.Infof("Public key: %s", encodedPublicKey)
					chainCode := ""
					if !isEdDSA {
						chainCodeBytes, err := mpcKeygenWrapper.KeyshareChainCode(result)
						if err != nil {
							t.logger.Error("fail to get chain code", "error", err)
							return "", "", err
						}
						chainCode = hex.EncodeToString(chainCodeBytes)
					}
					t.isKeygenFinished.Store(true)
					err = t.localStateAccessor.SaveLocalState(encodedPublicKey, encodedShare)
					return encodedPublicKey, chainCode, err
				}
			}
		}
	}
}

func (t *DKLSTssService) convertKeygenCommitteeToBytes(paries []string) ([]byte, error) {
	if len(paries) == 0 {
		return nil, fmt.Errorf("no parties provided")
	}
	result := make([]byte, 0)
	for _, party := range paries {
		result = append(result, []byte(party)...)
		result = append(result, byte(0))
	}
	// remove the last 0
	if len(result) > 0 {
		result = result[:len(result)-1]
	}
	return result, nil
}
