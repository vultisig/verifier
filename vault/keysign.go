package vault

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/sirupsen/logrus"
	vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"github.com/vultisig/mobile-tss-lib/tss"
	vcommon "github.com/vultisig/vultiserver/common"
	"github.com/vultisig/vultiserver/relay"
	vgrelay "github.com/vultisig/vultisig-go/relay"
	"golang.org/x/sync/errgroup"

	"github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

func (t *DKLSTssService) GetExistingVault(vaultFileName, password string) (*vaultType.Vault, error) {
	if vaultFileName == "" {
		return nil, fmt.Errorf("vault file name is empty")
	}
	content, err := t.storage.GetVault(vaultFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault file: %w", err)
	}
	vault, err := vcommon.DecryptVaultFromBackup(password, content)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault: %w", err)
	}
	return vault, nil
}

func (t *DKLSTssService) ProcessDKLSKeysign(req types.KeysignRequest) (map[string]tss.KeysignResponse, error) {
	result := map[string]tss.KeysignResponse{}
	vaultFileName := common.GetVaultBackupFilename(req.PublicKey, req.PluginID)
	vault, err := t.GetExistingVault(vaultFileName, t.cfg.EncryptionSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault: %w", err)
	}
	localStateAccessor := NewLocalStateAccessorImp(vault)
	t.localStateAccessor = localStateAccessor
	localPartyID := localStateAccessor.Vault.LocalPartyId
	relayClient := vgrelay.NewRelayClient(t.cfg.Relay.Server)
	if err := relayClient.RegisterSession(req.SessionID, localPartyID); err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	// wait longer for keysign start
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute+3*time.Second)
	defer cancel()

	partiesJoined, err := relayClient.WaitForSessionStart(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for session start: %w", err)
	}
	t.logger.WithFields(logrus.Fields{
		"session":        req.SessionID,
		"parties_joined": partiesJoined,
	}).Info("Session started")

	// start to do keysign
	for _, msg := range req.Messages {
		var publicKey string

		if msg.Chain.IsEdDSA() {
			publicKey = localStateAccessor.Vault.PublicKeyEddsa
		} else {
			publicKey = localStateAccessor.Vault.PublicKeyEcdsa
		}

		sig, err := t.keysignWithRetry(
			req.SessionID,
			req.HexEncryptionKey,
			publicKey,
			msg.Chain.IsEdDSA(),
			msg.Message,
			msg.Chain.GetDerivePath(),
			localPartyID,
			partiesJoined,
		)
		if err != nil {
			return result, fmt.Errorf("failed to keysign: %w", err)
		}
		if sig == nil {
			return result, fmt.Errorf("failed to keysign: signature is nil")
		}
		result[msg.Hash] = *sig
	}

	if err := relayClient.CompleteSession(req.SessionID, localPartyID); err != nil {
		t.logger.WithFields(logrus.Fields{
			"session": req.SessionID,
			"error":   err,
		}).Error("Failed to complete session")
	}

	return result, nil
}

func (t *DKLSTssService) keysign(sessionID string,
	hexEncryptionKey string,
	publicKey string,
	isEdDSA bool,
	message string,
	derivePath string,
	localPartyID string,
	keysignCommittee []string,
	attempt int) (*tss.KeysignResponse, error) {
	if publicKey == "" {
		return nil, fmt.Errorf("public key is empty")
	}
	if message == "" {
		return nil, fmt.Errorf("message is empty")
	}
	if derivePath == "" && !isEdDSA {
		return nil, fmt.Errorf("derive path is empty")
	}
	if localPartyID == "" {
		return nil, fmt.Errorf("local party id is empty")
	}
	if len(keysignCommittee) == 0 {
		return nil, fmt.Errorf("keysign committee is empty")
	}
	t.isKeysignFinished.Store(false)
	relayClient := vgrelay.NewRelayClient(t.cfg.Relay.Server)
	mpcWrapper := t.GetMPCKeygenWrapper(isEdDSA)
	t.logger.WithFields(logrus.Fields{
		"session_id":        sessionID,
		"public_key_ecdsa":  publicKey,
		"message":           message,
		"derive_path":       derivePath,
		"local_party_id":    localPartyID,
		"keysign_committee": keysignCommittee,
		"attempt":           attempt,
	}).Info("Keysign")

	md5Hash := md5.Sum([]byte(message))
	messageID := hex.EncodeToString(md5Hash[:])

	// we need to get the shares
	keyshare, err := t.localStateAccessor.GetLocalState(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyshare: %w", err)
	}
	keyshareBytes, err := base64.StdEncoding.DecodeString(keyshare)
	if err != nil {
		return nil, fmt.Errorf("failed to decode keyshare: %w", err)
	}
	keyshareHandle, err := mpcWrapper.KeyshareFromBytes(keyshareBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyshare from bytes: %w", err)
	}
	defer func() {
		if err := mpcWrapper.KeyshareFree(keyshareHandle); err != nil {
			t.logger.Error("failed to free keyshare", "error", err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var encryptedEncodedSetupMsg string
	if t.cfg.DoSetupMsg {
		id, e := mpcWrapper.KeyshareKeyID(keyshareHandle)
		if e != nil {
			return nil, fmt.Errorf("failed to get keyshare key ID: %w", e)
		}

		hashToSign, e := base64.StdEncoding.DecodeString(message)
		if e != nil {
			return nil, fmt.Errorf("failed to decode message: %w", e)
		}

		msg, e := mpcWrapper.SignSetupMsgNew(
			id,
			fmtDerivePath(derivePath),
			hashToSign,
			fmtIdsSlice(keysignCommittee),
		)
		if e != nil {
			return nil, fmt.Errorf("failed to create SignSetupMsgNew: %w", e)
		}

		payload, e := common.EncryptGCM(base64.StdEncoding.EncodeToString(msg), hexEncryptionKey)
		if e != nil {
			return nil, fmt.Errorf("failed to encrypt setup message: %w", e)
		}

		e = relayClient.UploadSetupMessage(sessionID, messageID, payload)
		if e != nil {
			return nil, fmt.Errorf("failed to relayClient.UploadSetupMessage: %w", e)
		}
		encryptedEncodedSetupMsg = payload
	} else {
		msg, e := relayClient.WaitForSetupMessage(ctx, sessionID, messageID)
		if e != nil {
			return nil, fmt.Errorf("failed to relayClient.WaitForSetupMessage: %w", e)
		}
		encryptedEncodedSetupMsg = msg
	}

	setupMsg, e := t.decodeDecryptMessage(encryptedEncodedSetupMsg, hexEncryptionKey)
	if e != nil {
		return nil, fmt.Errorf("failed to decodeDecryptMessage: %w", e)
	}

	setupHashToSign, err := mpcWrapper.DecodeMessage(setupMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to mpcWrapper.DecodeMessage: %w", err)
	}

	reqHashToSign, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}
	if !bytes.Equal(setupHashToSign, reqHashToSign) {
		return nil, fmt.Errorf("setupHashToSign is not equal to the reqHashToSign, stop keysign")
	}

	sessionHandle, err := mpcWrapper.SignSessionFromSetup(setupMsg, []byte(localPartyID), keyshareHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to SignSessionFromSetup: %w", err)
	}
	defer func() {
		err := mpcWrapper.SignSessionFree(sessionHandle)
		if err != nil {
			t.logger.Error("failed to free keysign session", "error", err)
		}
	}()

	eg := &errgroup.Group{}
	eg.Go(func() error {
		er := t.processKeysignOutbound(
			sessionHandle,
			sessionID,
			hexEncryptionKey,
			keysignCommittee,
			localPartyID,
			messageID,
			isEdDSA,
		)
		if er != nil {
			return fmt.Errorf("failed to processKeysignOutbound: %w", er)
		}
		return nil
	})
	eg.Go(func() error {
		er := t.processKeysignInbound(
			sessionHandle,
			sessionID,
			hexEncryptionKey,
			localPartyID,
			isEdDSA,
			messageID,
		)
		if er != nil {
			return fmt.Errorf("failed to processKeysignInbound: %w", er)
		}
		return nil
	})
	err = eg.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to process keysign: %w", err)
	}

	sig, err := mpcWrapper.SignSessionFinish(sessionHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to SignSessionFinish: %w", err)
	}

	t.logger.Infoln("keysign finished successfully")
	encodedKeysignResult := base64.StdEncoding.EncodeToString(sig)
	t.logger.Infof("keysign result: %s", encodedKeysignResult)

	t.logger.Infoln("keysign result is:", len(sig))
	rBytes := sig[:32]
	sBytes := sig[32:64]
	vBytes := sig[64:]
	derBytes, err := vcommon.GetDerSignature(rBytes, sBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get der signature: %w", err)
	}
	resp := &tss.KeysignResponse{
		Msg:          message,
		R:            hex.EncodeToString(sig[:32]),
		S:            hex.EncodeToString(sig[32:64]),
		DerSignature: hex.EncodeToString(derBytes),
		RecoveryID:   hex.EncodeToString(vBytes),
	}

	if t.cfg.DoSetupMsg {
		// no need to save same response from each consumer
		er := relayClient.MarkKeysignComplete(sessionID, messageID, *resp)
		if er != nil {
			return nil, fmt.Errorf("failed to save sig: %w", er)
		}
	}

	if isEdDSA {
		pubKeyBytes, err := hex.DecodeString(publicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key: %w", err)
		}

		if ed25519.Verify(pubKeyBytes, setupHashToSign, sig) {
			t.logger.Infoln("signature is valid")
		} else {
			t.logger.Error("signature is invalid")
		}
	} else {
		childPublicKey, err := mpcWrapper.KeyshareDeriveChildPublicKey(keyshareHandle, fmtDerivePath(derivePath))
		if err != nil {
			return nil, fmt.Errorf("failed to derive child public key: %w", err)
		}
		if len(sig) != 65 {
			return nil, fmt.Errorf("signature length is not 64")
		}
		recovery := sig[64]
		resp.RecoveryID = hex.EncodeToString([]byte{recovery})
		publicKeyECDSA, err := secp256k1.ParsePubKey(childPublicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}
		if ecdsa.Verify(publicKeyECDSA.ToECDSA(), setupHashToSign, new(big.Int).SetBytes(rBytes), new(big.Int).SetBytes(sBytes)) {
			t.logger.Infoln("signature is valid")
		} else {
			t.logger.Error("signature is invalid")
		}
	}
	return resp, nil
}

func (t *DKLSTssService) keysignWithRetry(sessionID string,
	hexEncryptionKey string,
	publicKey string,
	isEdDSA bool,
	message string,
	derivePath string,
	localPartyID string,
	keysignCommittee []string) (*tss.KeysignResponse, error) {
	for i := 0; i < 3; i++ {
		keysignResult, err := t.keysign(sessionID,
			hexEncryptionKey,
			publicKey,
			isEdDSA,
			message,
			derivePath,
			localPartyID,
			keysignCommittee, i)
		if err != nil {
			t.logger.WithFields(logrus.Fields{
				"session_id":        sessionID,
				"public_key_ecdsa":  publicKey,
				"message":           message,
				"derive_path":       derivePath,
				"local_party_id":    localPartyID,
				"keysign_committee": keysignCommittee,
				"attempt":           i,
			}).Error(err)
			time.Sleep(50 * time.Millisecond)
			continue
		} else {
			return keysignResult, nil
		}
	}
	return nil, fmt.Errorf("fail to keysign after max retry")
}

func (t *DKLSTssService) processKeysignOutbound(
	handle Handle,
	sessionID string,
	hexEncryptionKey string,
	parties []string,
	localPartyID string,
	messageID string,
	isEdDSA bool,
) error {
	messenger := relay.NewMessenger(t.cfg.Relay.Server, sessionID, hexEncryptionKey, true, messageID)
	mpcWrapper := t.GetMPCKeygenWrapper(isEdDSA)
	for {
		outbound, err := mpcWrapper.SignSessionOutputMessage(handle)
		if err != nil {
			t.logger.Error("failed to get output message", "error", err)
		}
		if len(outbound) == 0 {
			if t.isKeysignFinished.Load() {
				// we are finished
				return nil
			}
			time.Sleep(time.Millisecond * 100)
			continue
		}
		encodedOutbound := base64.StdEncoding.EncodeToString(outbound)
		for i := 0; i < len(parties); i++ {
			receiver, err := mpcWrapper.SignSessionMessageReceiver(handle, outbound, i)
			if err != nil {
				t.logger.Error("failed to get receiver message", "error", err)
			}
			if len(receiver) == 0 {
				continue
			}

			t.logger.Infoln("Sending message to", string(receiver))
			// send the message to the receiver
			if err := messenger.Send(localPartyID, string(receiver), encodedOutbound); err != nil {
				t.logger.Errorf("failed to send message: %v", err)
			}
		}
	}
}

func (t *DKLSTssService) processKeysignInbound(
	handle Handle,
	sessionID string,
	hexEncryptionKey string,
	localPartyID string,
	isEdDSA bool,
	messageID string,
) error {
	var messageCache sync.Map
	mpcWrapper := t.GetMPCKeygenWrapper(isEdDSA)
	relayClient := vgrelay.NewRelayClient(t.cfg.Relay.Server)
	start := time.Now()
	for {
		select {
		case <-time.After(time.Millisecond * 100):
			if time.Since(start) > time.Minute {
				t.isKeysignFinished.Store(true)
				return TssKeyGenTimeout
			}
			messages, err := relayClient.DownloadMessages(sessionID, localPartyID, messageID)
			if err != nil {
				t.logger.Error("fail to get messages", "error", err)
				continue
			}
			for _, message := range messages {
				if message.From == localPartyID {
					continue
				}
				cacheKey := fmt.Sprintf("%s-%s-%s", sessionID, localPartyID, message.Hash)
				if messageID != "" {
					cacheKey = fmt.Sprintf("%s-%s-%s-%s", sessionID, localPartyID, messageID, message.Hash)
				}
				if _, found := messageCache.Load(cacheKey); found {
					t.logger.Infof("Message already applied, skipping,hash: %s", message.Hash)
					continue
				}

				rawBody, err := t.decodeDecryptMessage(message.Body, hexEncryptionKey)
				if err != nil {
					t.logger.Error("fail to decode inbound message", "error", err)
					continue
				}
				// decode to get raw message
				t.logger.Infoln("Received message from", message.From)
				isFinished, err := mpcWrapper.SignSessionInputMessage(handle, rawBody)
				if err != nil {
					t.logger.Error("fail to apply input message", "error", err)
					continue
				}
				messageCache.Store(cacheKey, true)
				hashStr := message.Hash
				if err := relayClient.DeleteMessageFromServer(sessionID, localPartyID, hashStr, messageID); err != nil {
					t.logger.Error("fail to delete message", "error", err)
				}
				if isFinished {
					t.isKeysignFinished.Store(true)
					return nil
				}
			}
		}
	}
}

func fmtDerivePath(derivePath string) []byte {
	return []byte(strings.ReplaceAll(derivePath, "'", ""))
}

func fmtIdsSlice(ids []string) []byte {
	return []byte(strings.Join(ids, "\x00"))
}
