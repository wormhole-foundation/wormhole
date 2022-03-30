package gossipv1

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	vaa "github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"testing"

	"time"
)

func getVAA() vaa.VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	vaa := vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		ConsistencyLevel: uint8(32),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}

	return vaa
}

func TestSignedObservation(t *testing.T) {
	vaa := getVAA()
	digest := vaa.SigningMsg()

	txhash := []byte{97, 97, 97, 97, 97, 97}
	signature := []byte{97, 97, 97, 97, 97, 97}

	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	obsv := SignedObservation{
		Addr:      crypto.PubkeyToAddress(privKey.PublicKey).Bytes(),
		Hash:      digest.Bytes(),
		Signature: signature,
		TxHash:    txhash,
		MessageId: vaa.MessageID(),
	}

	assert.Equal(t, signature, obsv.Signature)
	assert.Equal(t, txhash, obsv.TxHash)
	assert.Equal(t, vaa.MessageID(), obsv.MessageId)
}

func getSignedObservation() SignedObservation {
	vaa := getVAA()
	digest := vaa.SigningMsg()

	txhash := []byte{97, 97, 97, 97, 97, 97}
	signature := []byte{97, 97, 97, 97, 97, 97}

	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	return SignedObservation{
		Addr:      crypto.PubkeyToAddress(privKey.PublicKey).Bytes(),
		Hash:      digest.Bytes(),
		Signature: signature,
		TxHash:    txhash,
		MessageId: vaa.MessageID(),
	}
}

func TestSignedObservation_String(t *testing.T) {
	obsv := getSignedObservation()
	obsvString := obsv.String()
	assert.Contains(t, obsvString, "addr")
	assert.Contains(t, obsvString, "hash")
	assert.Contains(t, obsvString, "signature")
	assert.Contains(t, obsvString, "tx_hash")
	assert.Contains(t, obsvString, "message_id")
	assert.Greater(t, len(obsvString), 200)
	assert.Less(t, len(obsvString), 300)
}

func TestSignedObservation_Descriptor(t *testing.T) {
	obsv := getSignedObservation()
	bytes, num := obsv.Descriptor()
	expected := "1f8b08000000000000ff016d0592fa0a16676f737369702f76312f676f737369702e70726f746f1209676f737369702e763122ee020a0d476f737369704d657373616765124d0a127369676e65645f6f62736572766174696f6e18022001280b321c2e676f737369702e76312e5369676e65644f62736572766174696f6e480052117369676e65644f62736572766174696f6e12470a107369676e65645f68656172746265617418032001280b321a2e676f737369702e76312e5369676e65644865617274626561744800520f7369676e656448656172746265617412550a167369676e65645f7661615f776974685f71756f72756d18042001280b321e2e676f737369702e76312e5369676e65645641415769746851756f72756d480052137369676e65645661615769746851756f72756d12630a1a7369676e65645f6f62736572766174696f6e5f7265717565737418052001280b32232e676f737369702e76312e5369676e65644f62736572766174696f6e52657175657374480052187369676e65644f62736572766174696f6e5265717565737442090a076d65737361676522720a0f5369676e6564486561727462656174121c0a0968656172746265617418012001280c5209686561727462656174121c0a097369676e617475726518022001280c52097369676e617475726512230a0d677561726469616e5f6164647218032001280c520c677561726469616e4164647222ff020a09486561727462656174121b0a096e6f64655f6e616d6518012001280952086e6f64654e616d6512180a07636f756e7465721802200128035207636f756e746572121c0a0974696d657374616d70180320012803520974696d657374616d7012380a086e6574776f726b7318042003280b321c2e676f737369702e76312e4865617274626561742e4e6574776f726b52086e6574776f726b7312180a0776657273696f6e180520012809520776657273696f6e12230a0d677561726469616e5f61646472180620012809520c677561726469616e4164647212250a0e626f6f745f74696d657374616d70180720012803520d626f6f7454696d657374616d701a7d0a074e6574776f726b120e0a02696418012001280d5202696412160a06686569676874180220012803520668656967687412290a10636f6e74726163745f61646472657373180320012809520f636f6e747261637441646472657373121f0a0b6572726f725f636f756e74180420012804520a6572726f72436f756e742291010a115369676e65644f62736572766174696f6e12120a046164647218012001280c52046164647212120a046861736818022001280c520468617368121c0a097369676e617475726518032001280c52097369676e617475726512170a0774785f6861736818042001280c5206747848617368121d0a0a6d6573736167655f696418052001280952096d657373616765496422270a135369676e65645641415769746851756f72756d12100a0376616118012001280c5203766161228e010a185369676e65644f62736572766174696f6e52657175657374122f0a136f62736572766174696f6e5f7265717565737418012001280c52126f62736572766174696f6e52657175657374121c0a097369676e617475726518022001280c52097369676e617475726512230a0d677561726469616e5f6164647218032001280c520c677561726469616e4164647222480a124f62736572766174696f6e5265717565737412190a08636861696e5f696418012001280d5207636861696e496412170a0774785f6861736818022001280c520674784861736842415a3f6769746875622e636f6d2f6365727475736f6e652f776f726d686f6c652f6e6f64652f706b672f70726f746f2f676f737369702f76313b676f737369707631620670726f746f338930f6b46d050000"

	assert.Equal(t, []int([]int{3}), num)
	assert.Equal(t, expected, hex.EncodeToString(bytes))
}

func TestSignedObservation_GetAddr(t *testing.T) {
	obsv := getSignedObservation()
	assert.Equal(t, obsv.Addr, obsv.GetAddr())
}

func TestSignedObservation_GetHash(t *testing.T) {
	obsv := getSignedObservation()
	assert.Equal(t, obsv.Hash, obsv.GetHash())
}

func TestSignedObservation_GetSignature(t *testing.T) {
	obsv := getSignedObservation()
	assert.Equal(t, obsv.Signature, obsv.GetSignature())
}

func TestSignedObservation_GetTxHash(t *testing.T) {
	obsv := getSignedObservation()
	assert.Equal(t, obsv.TxHash, obsv.GetTxHash())
}

func TestSignedObservation_GetMessageId(t *testing.T) {
	obsv := getSignedObservation()
	assert.Equal(t, obsv.MessageId, obsv.GetMessageId())
}
