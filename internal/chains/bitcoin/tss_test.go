package bitcoin

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/wire"
	"github.com/stretchr/testify/require"
	"github.com/vultisig/mobile-tss-lib/tss"
	"testing"
)

func TestTss_ComputeTxHash_Bitcoin_Witness(t *testing.T) {
	//// TODO Draft: implement BTC tx propose for plugins first, to correctly decode it here (maybe consider PSBT)
	return

	// https://www.blockchain.com/explorer/transactions/btc/bfce55eb0c07cb76adcb2b3f2e25402feace8ca90479bedcfebe3a380d1a9b59
	expectedTxHash := "bfce55eb0c07cb76adcb2b3f2e25402feace8ca90479bedcfebe3a380d1a9b59"

	txHex, err := unsign("02000000000103eb79db1a9e3ea2ab2b947f7af8d96e0eba49e35024860cd526cc4a6ad6aa184a0000000000fdffffff1730a4e5603e5a6eb8db8e7391580a2dd7b1c42005d7d356f4df171004544c3a0000000000fdffffffd8b790de36eec95e9d53625a6a905a598a5846397ec24aac10097f6fd80f7e4c0000000000fdffffff0236526c010000000017a914a05f068246b413fdab050738d4a8274a22f2614187000f450000000000160014902b3513ccbf3a48214c6e7aff8bcab1a904582902473044022029c43e0b75dbbef186f1e852bdb48e09024e1c061eb8804fe9d5601cdb8bd37c022001fcd6aceadf4d6c0b86863a3e3ff7e1bf151452112f88f78a0f72c1b5c081d8012102df4a8550dd9164271caf8e5124a594b27e2fd0423d530b4e2fecb650c049052302483045022100f127c2b2959853b5301c86b0adfced621a0f105fa553cea71619ce94c5f2546702207444aeb9dd86ce018d258b068c75e12782a26704a4a61aa2054d223ad84b49d9012103c5c477d19bc705e9050f3dae6f53da5afb0e8e6089d5b4c0dc84b50ad9c5f2e30247304402206a141f83ee8a518c08681fadb2c88ea99550697565893d660461154877eec12f022068db69180ada2e1dbf79bdcd3fa095dd58032fcbdcecc80c429932c56be2f222012103c5c477d19bc705e9050f3dae6f53da5afb0e8e6089d5b4c0dc84b50ad9c5f2e300000000")
	require.Nil(t, err, "unsign")

	txHash, err := NewTss().ComputeTxHash(
		txHex,
		[]tss.KeysignResponse{{
			R: "",
			S: "",
		}, {
			R: "",
			S: "",
		}, {
			R: "",
			S: "",
		}},
	)
	require.Nil(t, err, "NewTss().ComputeTxHash")
	require.Equal(t, expectedTxHash, txHash, "NewTss().ComputeTxHash")
}

// returns new txHex without sigs
func unsign(txHex string) (string, error) {
	data, err := hex.DecodeString(txHex)
	if err != nil {
		return "", fmt.Errorf("hex.DecodeString: %w", err)
	}

	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("tx.Deserialize: %w", err)
	}

	for i := range tx.TxIn {
		fmt.Println("Witness:", tx.TxIn[i].Witness.ToHexStrings())
		witnessPubKey := tx.TxIn[i].Witness[1]
		tx.TxIn[i].Witness = wire.TxWitness{nil, witnessPubKey}
		tx.TxIn[i].SignatureScript = nil
	}

	buf := &bytes.Buffer{}
	err = tx.Serialize(buf)
	if err != nil {
		return "", fmt.Errorf("tx.Serialize: %w", err)
	}
	return hex.EncodeToString(buf.Bytes()), nil
}
