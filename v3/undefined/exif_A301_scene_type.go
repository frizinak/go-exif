package exifundefined

import (
	"fmt"

	"encoding/binary"

	exifcommon "github.com/dsoprea/go-exif/v3/common"
	log "github.com/dsoprea/go-logging"
)

type TagExifA301SceneType uint8

func (TagExifA301SceneType) EncoderName() string {
	return "CodecExifA301SceneType"
}

func (st TagExifA301SceneType) String() string {
	return fmt.Sprintf("0x%02x", uint8(st))
}

const (
	TagUndefinedType_A301_SceneType_DirectlyPhotographedImage TagExifA301SceneType = 1
)

type CodecExifA301SceneType struct {
}

func (CodecExifA301SceneType) Encode(value interface{}, byteOrder binary.ByteOrder) (encoded []byte, unitCount uint32, err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	st, ok := value.(TagExifA301SceneType)
	if ok == false {
		log.Panicf("can only encode a TagExif9101ComponentsConfiguration")
	}

	ve := exifcommon.NewValueEncoder(byteOrder)

	ed, err := ve.Encode([]uint8{uint8(st)})
	log.PanicIf(err)

	// TODO(dustin): Confirm this size against the specification. It's non-specific about what type it is, but it looks to be no more than a single integer scalar. So, we're assuming it's a LONG.
	// nah... it's a byte

	return ed.Encoded, 1, nil
}

func (CodecExifA301SceneType) Decode(valueContext *exifcommon.ValueContext) (value EncodeableValue, err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	valueContext.SetUndefinedValueType(exifcommon.TypeByte)

	b, err := valueContext.ReadBytes()
	log.PanicIf(err)

	return TagExifA301SceneType(b[0]), nil
}

func init() {
	registerEncoder(
		TagExifA301SceneType(0),
		CodecExifA301SceneType{})

	registerDecoder(
		exifcommon.IfdExifStandardIfdIdentity.UnindexedString(),
		0xa301,
		CodecExifA301SceneType{})
}
