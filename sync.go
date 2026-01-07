package y_crdt

/*
 * Core Yjs defines two message types:
 * • YjsSyncStep1: Includes the State Set of the sending client. When received, the client should reply with YjsSyncStep2.
 * • YjsSyncStep2: Includes all missing structs and the complete delete set. When received, the client is assured that it
 *   received all information from the remote client.
 *
 * In a peer-to-peer network, you may want to introduce a SyncDone message type. Both parties should initiate the connection
 * with SyncStep1. When a client received SyncStep2, it should reply with SyncDone. When the local client received both
 * SyncStep2 and SyncDone, it is assured that it is synced to the remote client.
 *
 * In a client-server model, you want to handle this differently: The client should initiate the connection with SyncStep1.
 * When the server receives SyncStep1, it should reply with SyncStep2 immediately followed by SyncStep1. The client replies
 * with SyncStep2 when it receives SyncStep1. Optionally the server may send a SyncDone after it received SyncStep2, so the
 *  client knows that the sync is finished.  There are two reasons for this more elaborated sync model: 1. This protocol can
 * easily be implemented on top of http and websockets. 2. The server should only reply to requests, and not initiate them.
 * Therefore it is necesarry that the client initiates the sync.
 *
 * Construction of a message:
 * [messageType : varUint, message definition..]
 *
 * Note: A message does not include information about the room name. This must to be handled by the upper layer protocol!
 *
 * stringify[messageType] stringifies a message definition (messageType is already read from the bufffer)
 */

const (
	MessageYjsSyncStep1 = 0
	MessageYjsSyncStep2 = 1
	MessageYjsUpdate    = 2
)

// Create a sync step 1 message based on the state of the current shared document.
func WriteSyncStep1(encoder *UpdateEncoderV1, doc *Doc) {
	WriteVarUint(encoder.RestEncoder, MessageYjsSyncStep1)
	sv := EncodeStateVector(doc, nil, NewUpdateEncoderV1())
	WriteVarUint8Array(encoder.RestEncoder, sv)
}

func WriteSyncStep1FromUpdate(encoder *UpdateEncoderV1, update []uint8) {
	WriteVarUint(encoder.RestEncoder, MessageYjsSyncStep1)
	sv := EncodeStateVectorFromUpdate(update)
	WriteVarUint8Array(encoder.RestEncoder, sv)
}

func WriteSyncStep2(encoder *UpdateEncoderV1, doc *Doc, encodedStateVector []byte) {
	WriteVarUint(encoder.RestEncoder, MessageYjsSyncStep2)
	WriteVarUint8Array(encoder.RestEncoder, EncodeStateAsUpdate(doc, encodedStateVector))
}

func WriteSyncStep2FromUpdate(encoder *UpdateEncoderV1, update []byte, encodedStateVector []byte) {
	WriteVarUint(encoder.RestEncoder, MessageYjsSyncStep2)
	WriteVarUint8Array(encoder.RestEncoder, DiffUpdate(update, encodedStateVector))
}

// Read SyncStep1 message and reply with SyncStep2.
func ReadSyncStep1(decoder *UpdateDecoderV1, encoder *UpdateEncoderV1, doc *Doc) {
	data, err := ReadVarUint8Array(decoder.RestDecoder)
	if err != nil {
		Logf("[protocol] read sync step1 failed. err:%s", err.Error())
		return
	}

	WriteSyncStep2(encoder, doc, data.([]byte))
}

func ReadSyncStep2(decoder *UpdateDecoderV1, doc *Doc, transactionOrigin interface{}) {
	data, _ := ReadVarUint8Array(decoder.RestDecoder)
	ApplyUpdate(doc, data.([]byte), transactionOrigin)
}

func WriteUpdate(encoder *UpdateEncoderV1, update []byte) {
	WriteVarUint(encoder.RestEncoder, MessageYjsUpdate)
	WriteVarUint8Array(encoder.RestEncoder, update)
}

// ReadSyncMessage Read and apply Structs and then DeleteStore to a y instance.
func ReadSyncMessage(decoder *UpdateDecoderV1, encoder *UpdateEncoderV1, doc *Doc, transactionOrigin interface{}) int {
	messageType := ReadVarUint(decoder.RestDecoder)
	switch messageType {
	case MessageYjsSyncStep1:
		ReadSyncStep1(decoder, encoder, doc)
	case MessageYjsSyncStep2:
		ReadSyncStep2(decoder, doc, transactionOrigin)
	case MessageYjsUpdate:
		ReadSyncStep2(decoder, doc, transactionOrigin)
	default:

	}

	return int(messageType)
}
